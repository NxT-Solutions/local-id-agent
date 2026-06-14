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

struct AgentProcess(Mutex<Option<CommandChild>>);

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
    }
    Ok(path)
}

fn spawn_agent(app: &AppHandle, process: &AgentProcess) -> Result<(), String> {
    let config = ensure_config(app)?;
    let config_arg = config.to_string_lossy().to_string();

    if let Some(child) = process.0.lock().unwrap().take() {
        let _ = child.kill();
    }

    let (_event_rx, sidecar) = app
        .shell()
        .sidecar(SIDECAR_NAME)
        .map_err(|error| error.to_string())?
        .args(["--config", &config_arg])
        .spawn()
        .map_err(|error| error.to_string())?;

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
fn get_diagnostics(app: AppHandle, process: State<'_, AgentProcess>) -> Result<DiagnosticsInfo, String> {
    let config_path = ensure_config(&app)?.to_string_lossy().to_string();
    let sidecar_running = process.0.lock().unwrap().is_some();

    Ok(DiagnosticsInfo {
        app_version: app.package_info().version.to_string(),
        config_path,
        agent_url: "http://127.0.0.1:17443".to_string(),
        platform: std::env::consts::OS.to_string(),
        sidecar_running,
    })
}

fn show_main_window(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.show();
        let _ = window.set_focus();
    }
}

fn setup_tray(app: &AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    let open_item = MenuItem::with_id(app, "tray-open", "Open", true, None::<&str>)?;
    let restart_item =
        MenuItem::with_id(app, "tray-restart", "Restart Agent", true, None::<&str>)?;
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
        ])
        .run(tauri::generate_context!())
        .expect("error while running LocalID Agent desktop");
}
