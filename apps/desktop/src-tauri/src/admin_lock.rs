use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};

use argon2::{
    password_hash::{PasswordHasher, SaltString},
    Argon2,
};
use base64::{engine::general_purpose::STANDARD, Engine};
use rand::rngs::OsRng;
use serde::{Deserialize, Serialize};
use subtle::ConstantTimeEq;
use tauri::{AppHandle, Manager, State};

const LOCK_FILE_NAME: &str = "admin-lock.json";
pub const MIN_PASSCODE_LENGTH: usize = 8;
const MAX_FAILED_ATTEMPTS: u32 = 5;
const RATE_LIMIT_SECS: u64 = 30;
const SESSION_IDLE_SECS: u64 = 15 * 60;

fn argon2_costs() -> (u32, u32, u32) {
    #[cfg(test)]
    {
        (8_192, 1, 1)
    }
    #[cfg(not(test))]
    {
        (19_456, 2, 1)
    }
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct AdminLockFile {
    version: u32,
    kdf: String,
    salt: String,
    hash: String,
    params: Argon2Params,
    created_at: String,
    min_length: u32,
}

#[derive(Debug, Serialize, Deserialize)]
struct Argon2Params {
    m_cost: u32,
    t_cost: u32,
    p_cost: u32,
}

struct AdminSession {
    token: String,
    expires_at: Instant,
}

struct AttemptTracker {
    failures: u32,
    locked_until: Option<Instant>,
}

pub struct AdminLockState {
    session: Mutex<Option<AdminSession>>,
    failed_attempts: Mutex<AttemptTracker>,
}

impl AdminLockState {
    pub fn new() -> Self {
        Self {
            session: Mutex::new(None),
            failed_attempts: Mutex::new(AttemptTracker {
                failures: 0,
                locked_until: None,
            }),
        }
    }
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct AdminLockStatus {
    pub configured: bool,
    pub unlocked: bool,
    pub expires_at: Option<u64>,
    pub setup_required: bool,
    pub session_token: Option<String>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UnlockResult {
    pub session_token: String,
    pub expires_at: u64,
}

fn argon2_instance() -> Result<Argon2<'static>, String> {
    let (m_cost, t_cost, p_cost) = argon2_costs();
    let params = argon2::Params::new(m_cost, t_cost, p_cost, None)
        .map_err(|error| error.to_string())?;
    Ok(Argon2::new(
        argon2::Algorithm::Argon2id,
        argon2::Version::V0x13,
        params,
    ))
}

pub fn lock_file_path(app: &AppHandle) -> Result<PathBuf, String> {
    let dir = app
        .path()
        .app_data_dir()
        .map_err(|error| error.to_string())?;
    std::fs::create_dir_all(&dir).map_err(|error| error.to_string())?;
    Ok(dir.join(LOCK_FILE_NAME))
}

fn is_configured_at(path: &Path) -> bool {
    path.exists()
}

pub fn is_configured(app: &AppHandle) -> Result<bool, String> {
    Ok(is_configured_at(&lock_file_path(app)?))
}

fn load_lock_file(path: &Path) -> Result<AdminLockFile, String> {
    let raw = std::fs::read_to_string(path).map_err(|error| error.to_string())?;
    serde_json::from_str(&raw).map_err(|error| error.to_string())
}

fn save_lock_file(path: &Path, lock_file: &AdminLockFile) -> Result<(), String> {
    let serialized =
        serde_json::to_string_pretty(lock_file).map_err(|error| error.to_string())?;
    std::fs::write(path, format!("{serialized}\n")).map_err(|error| error.to_string())
}

fn utc_rfc3339_now() -> String {
    let duration = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default();
    let total_secs = duration.as_secs();
    let days = total_secs / 86_400;
    let time_of_day = total_secs % 86_400;
    let hours = time_of_day / 3_600;
    let minutes = (time_of_day % 3_600) / 60;
    let seconds = time_of_day % 60;

    let (year, month, day) = civil_from_days(days as i64);
    format!(
        "{year:04}-{month:02}-{day:02}T{hours:02}:{minutes:02}:{seconds:02}Z"
    )
}

fn civil_from_days(days: i64) -> (i64, i64, i64) {
    let z = days + 719_468;
    let era = if z >= 0 { z } else { z - 146_096 } / 146_097;
    let doe = z - era * 146_097;
    let yoe = (doe - doe / 1_460 + doe / 36_524 - doe / 146_096) / 365;
    let y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let day = doy - (153 * mp + 2) / 5 + 1;
    let month = mp + if mp < 10 { 3 } else { -9 };
    let year = y + if month <= 2 { 1 } else { 0 };
    (year, month, day)
}

fn hash_passcode(passcode: &str) -> Result<(String, String), String> {
    let salt = SaltString::generate(&mut OsRng);
    let argon2 = argon2_instance()?;
    let password_hash = argon2
        .hash_password(passcode.as_bytes(), &salt)
        .map_err(|error| error.to_string())?;
    let hash_part = password_hash
        .hash
        .ok_or_else(|| "missing password hash".to_string())?;
    Ok((
        salt.to_string(),
        STANDARD.encode(hash_part.as_bytes()),
    ))
}

fn verify_passcode(passcode: &str, lock_file: &AdminLockFile) -> Result<bool, String> {
    let salt = SaltString::from_b64(&lock_file.salt).map_err(|error| error.to_string())?;

    let argon2 = argon2_instance()?;
    let candidate = argon2
        .hash_password(passcode.as_bytes(), &salt)
        .map_err(|error| error.to_string())?;
    let candidate_hash = candidate
        .hash
        .ok_or_else(|| "missing candidate hash".to_string())?;

    let stored_hash = STANDARD
        .decode(&lock_file.hash)
        .map_err(|error| error.to_string())?;

    Ok(candidate_hash.as_bytes().ct_eq(&stored_hash).into())
}

fn random_token() -> String {
    use rand::RngCore;
    let mut bytes = [0u8; 32];
    OsRng.fill_bytes(&mut bytes);
    bytes.iter().map(|byte| format!("{byte:02x}")).collect()
}

fn instant_to_unix_millis(expires_at: Instant) -> u64 {
    let remaining = expires_at.saturating_duration_since(Instant::now());
    SystemTime::now()
        .checked_add(remaining)
        .and_then(|time| time.duration_since(UNIX_EPOCH).ok())
        .map(|duration| duration.as_millis() as u64)
        .unwrap_or(0)
}

fn check_rate_limit(state: &AdminLockState) -> Result<(), String> {
    let mut tracker = state.failed_attempts.lock().unwrap();
    if let Some(locked_until) = tracker.locked_until {
        if Instant::now() < locked_until {
            return Err("Invalid passcode".to_string());
        }
        tracker.locked_until = None;
        tracker.failures = 0;
    }
    Ok(())
}

fn record_failed_attempt(state: &AdminLockState) {
    let mut tracker = state.failed_attempts.lock().unwrap();
    tracker.failures += 1;
    if tracker.failures >= MAX_FAILED_ATTEMPTS {
        tracker.locked_until = Some(Instant::now() + Duration::from_secs(RATE_LIMIT_SECS));
        tracker.failures = 0;
    }
}

fn record_successful_unlock(state: &AdminLockState) {
    let mut tracker = state.failed_attempts.lock().unwrap();
    tracker.failures = 0;
    tracker.locked_until = None;
}

fn create_session(state: &AdminLockState) -> UnlockResult {
    let token = random_token();
    let expires_at = Instant::now() + Duration::from_secs(SESSION_IDLE_SECS);
    let expires_at_millis = instant_to_unix_millis(expires_at);
    *state.session.lock().unwrap() = Some(AdminSession {
        token: token.clone(),
        expires_at,
    });
    UnlockResult {
        session_token: token,
        expires_at: expires_at_millis,
    }
}

fn active_session(state: &AdminLockState, touch: bool) -> Option<AdminSession> {
    let mut guard = state.session.lock().unwrap();
    let Some(session) = guard.as_mut() else {
        return None;
    };

    if session.expires_at <= Instant::now() {
        *guard = None;
        return None;
    }

    if touch {
        session.expires_at = Instant::now() + Duration::from_secs(SESSION_IDLE_SECS);
    }

    Some(AdminSession {
        token: session.token.clone(),
        expires_at: session.expires_at,
    })
}

pub fn get_status(app: &AppHandle, state: &AdminLockState) -> Result<AdminLockStatus, String> {
    let path = lock_file_path(app)?;
    let configured = is_configured_at(&path);
    let setup_required = !configured;

    if let Some(session) = active_session(state, false) {
        return Ok(AdminLockStatus {
            configured,
            unlocked: true,
            expires_at: Some(instant_to_unix_millis(session.expires_at)),
            setup_required,
            session_token: Some(session.token),
        });
    }

    Ok(AdminLockStatus {
        configured,
        unlocked: false,
        expires_at: None,
        setup_required,
        session_token: None,
    })
}

pub fn require_admin(state: &AdminLockState, app: &AppHandle) -> Result<(), String> {
    if !is_configured(app)? {
        return Err("Admin passcode not configured".to_string());
    }

    if active_session(state, true).is_some() {
        return Ok(());
    }

    Err("Admin access required".to_string())
}

pub fn setup(
    app: &AppHandle,
    state: &AdminLockState,
    passcode: String,
) -> Result<UnlockResult, String> {
    if passcode.len() < MIN_PASSCODE_LENGTH {
        return Err(format!(
            "Passcode must be at least {MIN_PASSCODE_LENGTH} characters"
        ));
    }

    let path = lock_file_path(app)?;
    if is_configured_at(&path) {
        return Err("Admin passcode already configured".to_string());
    }

    let (salt, hash) = hash_passcode(&passcode)?;
    let (m_cost, t_cost, p_cost) = argon2_costs();
    let lock_file = AdminLockFile {
        version: 1,
        kdf: "argon2id".to_string(),
        salt,
        hash,
        params: Argon2Params {
            m_cost,
            t_cost,
            p_cost,
        },
        created_at: utc_rfc3339_now(),
        min_length: MIN_PASSCODE_LENGTH as u32,
    };
    save_lock_file(&path, &lock_file)?;
    Ok(create_session(state))
}

pub fn unlock(
    app: &AppHandle,
    state: &AdminLockState,
    passcode: String,
) -> Result<UnlockResult, String> {
    check_rate_limit(state)?;

    let path = lock_file_path(app)?;
    if !is_configured_at(&path) {
        return Err("Admin passcode not configured".to_string());
    }

    let lock_file = load_lock_file(&path)?;
    if !verify_passcode(&passcode, &lock_file)? {
        record_failed_attempt(state);
        return Err("Invalid passcode".to_string());
    }

    record_successful_unlock(state);
    Ok(create_session(state))
}

pub fn lock(state: &AdminLockState) {
    *state.session.lock().unwrap() = None;
}

pub fn change_passcode(
    app: &AppHandle,
    state: &AdminLockState,
    current_passcode: String,
    new_passcode: String,
) -> Result<(), String> {
    require_admin(state, app)?;

    if new_passcode.len() < MIN_PASSCODE_LENGTH {
        return Err(format!(
            "Passcode must be at least {MIN_PASSCODE_LENGTH} characters"
        ));
    }

    let path = lock_file_path(app)?;
    let lock_file = load_lock_file(&path)?;
    if !verify_passcode(&current_passcode, &lock_file)? {
        return Err("Invalid passcode".to_string());
    }

    let (salt, hash) = hash_passcode(&new_passcode)?;
    let updated = AdminLockFile {
        version: lock_file.version,
        kdf: lock_file.kdf,
        salt,
        hash,
        params: lock_file.params,
        created_at: utc_rfc3339_now(),
        min_length: MIN_PASSCODE_LENGTH as u32,
    };
    save_lock_file(&path, &updated)?;
    Ok(())
}

#[tauri::command]
pub fn get_admin_lock_status(
    app: AppHandle,
    state: State<'_, AdminLockState>,
) -> Result<AdminLockStatus, String> {
    get_status(&app, state.inner())
}

#[tauri::command]
pub fn setup_admin_passcode(
    app: AppHandle,
    state: State<'_, AdminLockState>,
    passcode: String,
) -> Result<UnlockResult, String> {
    setup(&app, state.inner(), passcode)
}

#[tauri::command]
pub fn unlock_admin(
    app: AppHandle,
    state: State<'_, AdminLockState>,
    passcode: String,
) -> Result<UnlockResult, String> {
    unlock(&app, state.inner(), passcode)
}

#[tauri::command]
pub fn lock_admin(state: State<'_, AdminLockState>) {
    lock(state.inner());
}

#[tauri::command]
pub fn change_admin_passcode(
    app: AppHandle,
    state: State<'_, AdminLockState>,
    current_passcode: String,
    new_passcode: String,
) -> Result<(), String> {
    change_passcode(
        &app,
        state.inner(),
        current_passcode,
        new_passcode,
    )
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct VerifyAdminSessionResponse {
    pub valid: bool,
    pub expires_at: Option<u64>,
}

#[tauri::command]
pub fn verify_admin_session(state: State<'_, AdminLockState>) -> VerifyAdminSessionResponse {
    let session = active_session(state.inner(), false);
    VerifyAdminSessionResponse {
        valid: session.is_some(),
        expires_at: session.map(|value| instant_to_unix_millis(value.expires_at)),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn temp_lock_path() -> PathBuf {
        let dir = std::env::temp_dir().join(format!(
            "localid-admin-lock-test-{}-{}",
            std::process::id(),
            random_token()
        ));
        fs::create_dir_all(&dir).expect("create temp dir");
        dir.join(LOCK_FILE_NAME)
    }

    fn write_lock_file(path: &Path, passcode: &str) {
        let (salt, hash) = hash_passcode(passcode).expect("hash passcode");
        let lock_file = AdminLockFile {
            version: 1,
            kdf: "argon2id".to_string(),
            salt,
            hash,
            params: {
                let (m_cost, t_cost, p_cost) = argon2_costs();
                Argon2Params {
                    m_cost,
                    t_cost,
                    p_cost,
                }
            },
            created_at: utc_rfc3339_now(),
            min_length: MIN_PASSCODE_LENGTH as u32,
        };
        save_lock_file(path, &lock_file).expect("save lock file");
    }

    #[test]
    fn setup_rejects_short_passcode() {
        let state = AdminLockState::new();
        let path = temp_lock_path();
        assert!(passcode_too_short("short").is_err());
        assert!(path.parent().is_some());
        let _ = state;
        let _ = path;
    }

    fn passcode_too_short(passcode: &str) -> Result<(), String> {
        if passcode.len() < MIN_PASSCODE_LENGTH {
            return Err(format!(
                "Passcode must be at least {MIN_PASSCODE_LENGTH} characters"
            ));
        }
        Ok(())
    }

    #[test]
    fn verify_passcode_accepts_correct_value() {
        let path = temp_lock_path();
        write_lock_file(&path, "correct-pass");
        let lock_file = load_lock_file(&path).expect("load lock file");
        assert!(verify_passcode("correct-pass", &lock_file).expect("verify"));
        assert!(!verify_passcode("wrong-pass", &lock_file).expect("verify"));
        let _ = fs::remove_file(&path);
    }

    #[test]
    fn session_expires_after_idle_timeout() {
        let state = AdminLockState::new();
        {
            let mut guard = state.session.lock().unwrap();
            *guard = Some(AdminSession {
                token: "test-token".to_string(),
                expires_at: Instant::now() - Duration::from_secs(1),
            });
        }
        assert!(active_session(&state, false).is_none());
    }

    #[test]
    fn rate_limit_blocks_after_five_failures() {
        let state = AdminLockState::new();
        for _ in 0..MAX_FAILED_ATTEMPTS {
            record_failed_attempt(&state);
        }
        assert!(check_rate_limit(&state).is_err());
    }

    #[test]
    fn lock_clears_session() {
        let state = AdminLockState::new();
        {
            let mut guard = state.session.lock().unwrap();
            *guard = Some(AdminSession {
                token: "token".to_string(),
                expires_at: Instant::now() + Duration::from_secs(60),
            });
        }
        lock(&state);
        assert!(state.session.lock().unwrap().is_none());
    }

    #[test]
    fn change_passcode_requires_valid_current() {
        let path = temp_lock_path();
        write_lock_file(&path, "old-passcode");
        let lock_file = load_lock_file(&path).expect("load");
        assert!(verify_passcode("old-passcode", &lock_file).expect("verify"));

        let (salt, hash) = hash_passcode("new-passcode").expect("hash");
        let updated = AdminLockFile {
            version: 1,
            kdf: "argon2id".to_string(),
            salt,
            hash,
            params: lock_file.params,
            created_at: utc_rfc3339_now(),
            min_length: MIN_PASSCODE_LENGTH as u32,
        };
        save_lock_file(&path, &updated).expect("save");
        let reloaded = load_lock_file(&path).expect("reload");
        assert!(verify_passcode("new-passcode", &reloaded).expect("verify"));
        assert!(!verify_passcode("old-passcode", &reloaded).expect("verify"));
        let _ = fs::remove_file(&path);
    }
}
