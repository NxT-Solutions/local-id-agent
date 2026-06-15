use std::path::PathBuf;
use std::sync::Mutex;

use serde::Serialize;
use tauri::{
    menu::{Menu, MenuItem},
    tray::TrayIconBuilder,
    AppHandle, Manager, State,
};
use tauri_plugin_shell::process::CommandChild;
use tauri_plugin_shell::ShellExt;

const SIDECAR_NAME: &str = "localid-agent";
const AGENT_BASE_URL: &str = "http://127.0.0.1:17443";
const AGENT_FETCH_TIMEOUT_SECS: u64 = 4;
const AGENT_FETCH_SIGN_TIMEOUT_SECS: u64 = 120;
const REQUIRED_ALLOWED_ORIGINS: [&str; 2] = ["tauri://localhost", "http://localhost:1420"];
const REQUIRED_ALLOWED_BACKENDS: [&str; 1] = ["http://localhost:8000"];
const FALLBACK_PROVIDER: &str = "mock";

struct AgentProcess(Mutex<Option<CommandChild>>);

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct AgentFetchResponse {
    status: u16,
    body: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct DiagnosticsInfo {
    app_version: String,
    config_path: String,
    agent_url: String,
    platform: String,
    sidecar_running: bool,
}

fn config_path(app: &AppHandle) -> Result<PathBuf, String> {
    let dir = app
        .path()
        .app_data_dir()
        .map_err(|error| error.to_string())?;
    std::fs::create_dir_all(&dir).map_err(|error| error.to_string())?;
    Ok(dir.join("config.json"))
}

fn ensure_config(app: &AppHandle) -> Result<PathBuf, String> {
    let path = config_path(app)?;
    if !path.exists() {
        let template = include_str!("../config.desktop.json");
        std::fs::write(&path, template).map_err(|error| error.to_string())?;
    } else {
        backfill_desktop_config(&path)?;
    }
    Ok(path)
}

fn backfill_desktop_config(path: &PathBuf) -> Result<(), String> {
    let raw = std::fs::read_to_string(path).map_err(|error| error.to_string())?;
    let mut root: serde_json::Value =
        serde_json::from_str(&raw).map_err(|error| error.to_string())?;

    let Some(root_object) = root.as_object_mut() else {
        return Ok(());
    };

    let security_value = root_object
        .entry("security")
        .or_insert_with(|| serde_json::json!({}));
    if !security_value.is_object() {
        return Ok(());
    }

    let Some(security) = security_value.as_object_mut() else {
        return Ok(());
    };

    let origins_changed = {
        let origins_value = security
            .entry("allowed_origins")
            .or_insert_with(|| serde_json::json!([]));
        ensure_values_in_array(origins_value, &REQUIRED_ALLOWED_ORIGINS)
    };
    let backends_changed = {
        let backends_value = security
            .entry("allowed_backends")
            .or_insert_with(|| serde_json::json!([]));
        ensure_values_in_array(backends_value, &REQUIRED_ALLOWED_BACKENDS)
    };
    let provider_changed = ensure_valid_default_provider(root_object);

    if !origins_changed && !backends_changed && !provider_changed {
        return Ok(());
    }

    let normalized = serde_json::to_string_pretty(&root).map_err(|error| error.to_string())?;
    std::fs::write(path, format!("{normalized}\n")).map_err(|error| error.to_string())?;

    Ok(())
}

fn ensure_valid_default_provider(
    root_object: &mut serde_json::Map<String, serde_json::Value>,
) -> bool {
    let providers_value = root_object
        .entry("providers")
        .or_insert_with(|| serde_json::json!({}));
    if !providers_value.is_object() {
        *providers_value = serde_json::json!({});
    }

    let Some(providers) = providers_value.as_object_mut() else {
        return false;
    };

    let configured_default = providers
        .get("default")
        .and_then(|value| value.as_str())
        .unwrap_or(FALLBACK_PROVIDER)
        .to_string();
    let mut changed = false;

    let default_provider = match configured_default.as_str() {
        "mock" | "pkcs11" | "belgian_eid" => configured_default,
        _ => {
            changed = true;
            FALLBACK_PROVIDER.to_string()
        }
    };

    if providers.get("default").and_then(|value| value.as_str()) != Some(default_provider.as_str())
    {
        providers.insert(
            "default".to_string(),
            serde_json::Value::String(default_provider.clone()),
        );
        changed = true;
    }

    let selected_provider_value = providers
        .entry(default_provider.clone())
        .or_insert_with(|| serde_json::json!({}));
    if !selected_provider_value.is_object() {
        *selected_provider_value = serde_json::json!({});
        changed = true;
    }

    let default_enabled = selected_provider_value
        .as_object()
        .and_then(|value| value.get("enabled"))
        .and_then(|value| value.as_bool())
        .unwrap_or(false);

    if let Some(selected_provider) = selected_provider_value.as_object_mut() {
        if !default_enabled {
            selected_provider.insert("enabled".to_string(), serde_json::Value::Bool(true));
            changed = true;
        }
    }

    changed
}

fn with_sidecar_rebuild_hint(error: impl ToString) -> String {
    let message = error.to_string();
    let lowercase = message.to_lowercase();
    let looks_like_missing_sidecar = lowercase.contains("sidecar")
        && (lowercase.contains("not found")
            || lowercase.contains("no such file")
            || lowercase.contains("could not find"));

    if !looks_like_missing_sidecar {
        return message;
    }

    format!(
        "{message}. If the sidecar binary is missing or stale, run `pnpm run build:sidecar` from the repo root."
    )
}

fn ensure_values_in_array(value: &mut serde_json::Value, required: &[&str]) -> bool {
    if !value.is_array() {
        *value = serde_json::json!([]);
    }

    let Some(array) = value.as_array_mut() else {
        return false;
    };

    let mut changed = false;
    for item in required {
        let exists = array
            .iter()
            .any(|existing| existing.as_str().is_some_and(|text| text == *item));
        if !exists {
            array.push(serde_json::Value::String((*item).to_string()));
            changed = true;
        }
    }

    changed
}

fn spawn_agent(app: &AppHandle, process: &AgentProcess) -> Result<(), String> {
    let config = ensure_config(app)?;
    let config_arg = config.to_string_lossy().to_string();
    let pkcs11_pin = std::env::var("LOCALID_PKCS11_PIN")
        .ok()
        .filter(|pin| !pin.trim().is_empty());
    let beid_module = std::env::var("LOCALID_BEID_PKCS11_MODULE")
        .ok()
        .map(|module| module.trim().to_string())
        .filter(|module| !module.is_empty())
        .filter(|module| std::path::Path::new(module).is_file());

    if let Some(child) = process.0.lock().unwrap().take() {
        let _ = child.kill();
    }

    let mut sidecar_command = app
        .shell()
        .sidecar(SIDECAR_NAME)
        .map_err(with_sidecar_rebuild_hint)?;

    sidecar_command = sidecar_command.args(["--config", &config_arg]);
    if let Some(pin) = pkcs11_pin {
        sidecar_command = sidecar_command.env("LOCALID_PKCS11_PIN", pin);
    }
    if let Some(module) = beid_module {
        sidecar_command = sidecar_command.env("LOCALID_BEID_PKCS11_MODULE", module);
    }

    let (_event_rx, sidecar) = sidecar_command.spawn().map_err(with_sidecar_rebuild_hint)?;

    *process.0.lock().unwrap() = Some(sidecar);
    Ok(())
}

fn stop_agent(process: &AgentProcess) -> Result<(), String> {
    if let Some(child) = process.0.lock().unwrap().take() {
        child.kill().map_err(|error| error.to_string())?;
    }
    Ok(())
}

#[tauri::command]
fn get_config_path(app: AppHandle) -> Result<String, String> {
    Ok(ensure_config(&app)?.to_string_lossy().to_string())
}

#[tauri::command]
fn read_config(app: AppHandle) -> Result<String, String> {
    let path = ensure_config(&app)?;
    std::fs::read_to_string(path).map_err(|error| error.to_string())
}

#[tauri::command]
fn write_config(app: AppHandle, contents: String) -> Result<(), String> {
    let _: serde_json::Value =
        serde_json::from_str(&contents).map_err(|error| error.to_string())?;
    let path = ensure_config(&app)?;
    std::fs::write(path, contents).map_err(|error| error.to_string())
}

#[tauri::command]
fn restart_agent(app: AppHandle, process: State<'_, AgentProcess>) -> Result<(), String> {
    spawn_agent(&app, &process)
}

#[tauri::command]
fn get_diagnostics(
    app: AppHandle,
    process: State<'_, AgentProcess>,
) -> Result<DiagnosticsInfo, String> {
    let config_path = ensure_config(&app)?.to_string_lossy().to_string();
    let sidecar_running = process.0.lock().unwrap().is_some();

    Ok(DiagnosticsInfo {
        app_version: app.package_info().version.to_string(),
        config_path,
        agent_url: AGENT_BASE_URL.to_string(),
        platform: std::env::consts::OS.to_string(),
        sidecar_running,
    })
}

fn agent_fetch_timeout(method: &str, path: &str) -> std::time::Duration {
    if method.eq_ignore_ascii_case("POST") && path.starts_with("/sign-challenge") {
        std::time::Duration::from_secs(AGENT_FETCH_SIGN_TIMEOUT_SECS)
    } else {
        std::time::Duration::from_secs(AGENT_FETCH_TIMEOUT_SECS)
    }
}

#[tauri::command]
fn agent_fetch(
    method: String,
    path: String,
    body: Option<String>,
    origin: Option<String>,
) -> Result<AgentFetchResponse, String> {
    if !path.starts_with('/') {
        return Err("agent path must start with '/'".to_string());
    }

    let url = format!("{AGENT_BASE_URL}{path}");
    let timeout = agent_fetch_timeout(&method, &path);
    let agent = ureq::AgentBuilder::new().timeout(timeout).build();

    let upper_method = method.to_ascii_uppercase();
    let response = match upper_method.as_str() {
        "GET" => agent.get(&url).call(),
        "POST" => {
            let mut request = agent.post(&url).set("Content-Type", "application/json");
            if let Some(origin_value) = origin.as_deref().filter(|value| !value.is_empty()) {
                request = request.set("Origin", origin_value);
            }

            match body {
                Some(payload) => request.send_string(&payload),
                None => request.call(),
            }
        }
        _ => return Err(format!("unsupported HTTP method: {method}")),
    };

    match response {
        Ok(resp) => {
            let status = resp.status();
            let body = resp.into_string().map_err(|error| error.to_string())?;
            Ok(AgentFetchResponse { status, body })
        }
        Err(ureq::Error::Status(status, resp)) => {
            let body = resp.into_string().unwrap_or_default();
            Ok(AgentFetchResponse { status, body })
        }
        Err(error) => Err(error.to_string()),
    }
}

fn show_main_window(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.show();
        let _ = window.set_focus();
    }
}

fn setup_tray(app: &AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    let open_item = MenuItem::with_id(app, "tray-open", "Open", true, None::<&str>)?;
    let restart_item = MenuItem::with_id(app, "tray-restart", "Restart Agent", true, None::<&str>)?;
    let quit_item = MenuItem::with_id(app, "tray-quit", "Quit", true, None::<&str>)?;
    let menu = Menu::with_items(app, &[&open_item, &restart_item, &quit_item])?;

    let _tray = TrayIconBuilder::new()
        .menu(&menu)
        .tooltip("LocalID Agent")
        .on_menu_event(|app, event| match event.id.as_ref() {
            "tray-open" => show_main_window(app),
            "tray-restart" => {
                if let Some(process) = app.try_state::<AgentProcess>() {
                    let _ = spawn_agent(app, process.inner());
                }
            }
            "tray-quit" => {
                if let Some(process) = app.try_state::<AgentProcess>() {
                    let _ = stop_agent(process.inner());
                }
                app.exit(0);
            }
            _ => {}
        })
        .build(app)?;

    Ok(())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_http::init())
        .plugin(tauri_plugin_shell::init())
        .manage(AgentProcess(Mutex::new(None)))
        .setup(|app| {
            setup_tray(app.handle())?;

            if let Some(process) = app.try_state::<AgentProcess>() {
                spawn_agent(app.handle(), process.inner())?;
            }

            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                let _ = window.hide();
            }
        })
        .invoke_handler(tauri::generate_handler![
            get_config_path,
            read_config,
            write_config,
            restart_agent,
            get_diagnostics,
            agent_fetch,
        ])
        .run(tauri::generate_context!())
        .expect("error while running LocalID Agent desktop");
}
