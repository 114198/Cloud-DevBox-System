//! Media service configuration

use serde::Deserialize;

#[derive(Debug, Deserialize)]
pub struct MediaConfig {
    pub host: String,
    pub port: u16,
    pub webrtc: WebRTCConfig,
    pub recording: RecordingConfig,
}

#[derive(Debug, Deserialize)]
pub struct WebRTCConfig {
    pub stun_servers: Vec<String>,
    pub turn_servers: Vec<TurnServer>,
    pub max_video_bitrate: u32,
    pub max_audio_bitrate: u32,
}

#[derive(Debug, Deserialize)]
pub struct TurnServer {
    pub url: String,
    pub username: String,
    pub credential: String,
}

#[derive(Debug, Deserialize)]
pub struct RecordingConfig {
    pub enabled: bool,
    pub format: String,
    pub storage_path: String,
}

impl Default for MediaConfig {
    fn default() -> Self {
        Self {
            host: "0.0.0.0".to_string(),
            port: 8084,
            webrtc: WebRTCConfig {
                stun_servers: vec!["stun:stun.l.google.com:19302".to_string()],
                turn_servers: vec![],
                max_video_bitrate: 2_500_000,
                max_audio_bitrate: 128_000,
            },
            recording: RecordingConfig {
                enabled: false,
                format: "mp4".to_string(),
                storage_path: "/var/lib/devbox/recordings".to_string(),
            },
        }
    }
}
