//! Data models for the media service

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// Video quality levels
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum VideoQuality {
    #[serde(rename = "1080p")]
    Quality1080p,
    #[serde(rename = "720p")]
    Quality720p,
    #[serde(rename = "480p")]
    Quality480p,
    #[serde(rename = "360p")]
    Quality360p,
    AudioOnly,
}

impl Default for VideoQuality {
    fn default() -> Self {
        Self::Quality720p
    }
}

impl VideoQuality {
    /// Get the maximum bitrate for this quality level in bps
    pub fn max_bitrate(&self) -> u32 {
        match self {
            Self::Quality1080p => 2_500_000,
            Self::Quality720p => 1_500_000,
            Self::Quality480p => 800_000,
            Self::Quality360p => 400_000,
            Self::AudioOnly => 0,
        }
    }

    /// Get the resolution for this quality level
    pub fn resolution(&self) -> (u32, u32) {
        match self {
            Self::Quality1080p => (1920, 1080),
            Self::Quality720p => (1280, 720),
            Self::Quality480p => (854, 480),
            Self::Quality360p => (640, 360),
            Self::AudioOnly => (0, 0),
        }
    }
}

/// Media track type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum MediaTrackKind {
    Audio,
    Video,
}

/// Media track state
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MediaTrack {
    pub id: String,
    pub kind: MediaTrackKind,
    pub enabled: bool,
    pub muted: bool,
    pub label: Option<String>,
}

/// Participant in a media room
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MediaParticipant {
    pub id: Uuid,
    pub user_id: Uuid,
    pub display_name: String,
    pub audio_track: Option<MediaTrack>,
    pub video_track: Option<MediaTrack>,
    pub screen_track: Option<MediaTrack>,
    pub video_quality: VideoQuality,
    pub joined_at: DateTime<Utc>,
}

/// Media room representing a meeting's media session
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MediaRoom {
    pub id: Uuid,
    pub meeting_id: Uuid,
    pub participants: Vec<MediaParticipant>,
    pub max_participants: usize,
    pub created_at: DateTime<Utc>,
    pub recording_enabled: bool,
}

/// SFU transport configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransportConfig {
    pub id: String,
    pub ice_parameters: IceParameters,
    pub ice_candidates: Vec<IceCandidate>,
    pub dtls_parameters: DtlsParameters,
}

/// ICE parameters for WebRTC connection
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct IceParameters {
    pub username_fragment: String,
    pub password: String,
    pub ice_lite: bool,
}

/// ICE candidate
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct IceCandidate {
    pub foundation: String,
    pub priority: u32,
    pub ip: String,
    pub protocol: String,
    pub port: u16,
    pub r#type: String,
}

/// DTLS parameters for secure connection
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DtlsParameters {
    pub role: String,
    pub fingerprints: Vec<DtlsFingerprint>,
}

/// DTLS fingerprint
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DtlsFingerprint {
    pub algorithm: String,
    pub value: String,
}


/// RTP parameters for media production
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RtpParameters {
    pub codecs: Vec<RtpCodec>,
    pub header_extensions: Vec<RtpHeaderExtension>,
    pub encodings: Vec<RtpEncoding>,
}

/// RTP codec configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RtpCodec {
    pub mime_type: String,
    pub payload_type: u8,
    pub clock_rate: u32,
    pub channels: Option<u8>,
    pub parameters: Option<serde_json::Value>,
}

/// RTP header extension
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RtpHeaderExtension {
    pub uri: String,
    pub id: u8,
    pub encrypt: bool,
}

/// RTP encoding parameters
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RtpEncoding {
    pub ssrc: Option<u32>,
    pub rid: Option<String>,
    pub max_bitrate: Option<u32>,
    pub scale_resolution_down_by: Option<f32>,
}

/// Request to create a transport
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateTransportRequest {
    pub room_id: Uuid,
    pub participant_id: Uuid,
    pub direction: TransportDirection,
}

/// Transport direction
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TransportDirection {
    Send,
    Recv,
}

/// Response for transport creation
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateTransportResponse {
    pub transport_id: String,
    pub ice_parameters: IceParameters,
    pub ice_candidates: Vec<IceCandidate>,
    pub dtls_parameters: DtlsParameters,
}

/// Request to connect a transport
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ConnectTransportRequest {
    pub transport_id: String,
    pub dtls_parameters: DtlsParameters,
}

/// Request to produce media
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ProduceRequest {
    pub transport_id: String,
    pub kind: MediaTrackKind,
    pub rtp_parameters: RtpParameters,
    pub app_data: Option<serde_json::Value>,
}

/// Response for media production
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ProduceResponse {
    pub producer_id: String,
}

/// Request to consume media
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ConsumeRequest {
    pub transport_id: String,
    pub producer_id: String,
    pub rtp_capabilities: RtpCapabilities,
}

/// RTP capabilities
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RtpCapabilities {
    pub codecs: Vec<RtpCodec>,
    pub header_extensions: Vec<RtpHeaderExtension>,
}

/// Response for media consumption
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ConsumeResponse {
    pub consumer_id: String,
    pub producer_id: String,
    pub kind: MediaTrackKind,
    pub rtp_parameters: RtpParameters,
}

/// Quality adaptation event
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct QualityAdaptationEvent {
    pub participant_id: Uuid,
    pub previous_quality: VideoQuality,
    pub new_quality: VideoQuality,
    pub reason: QualityChangeReason,
    pub bandwidth_estimate: Option<u32>,
}

/// Reason for quality change
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum QualityChangeReason {
    BandwidthLow,
    BandwidthRecovered,
    UserRequested,
    CpuOverload,
    PacketLoss,
}

/// Media statistics
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MediaStats {
    pub participant_id: Uuid,
    pub audio_bitrate: Option<u32>,
    pub video_bitrate: Option<u32>,
    pub packet_loss: f32,
    pub jitter: f32,
    pub round_trip_time: f32,
    pub timestamp: DateTime<Utc>,
}

/// Error response
#[derive(Debug, Serialize)]
pub struct ErrorResponse {
    pub code: String,
    pub message: String,
    pub details: Option<String>,
}

impl ErrorResponse {
    pub fn new(code: impl Into<String>, message: impl Into<String>) -> Self {
        Self {
            code: code.into(),
            message: message.into(),
            details: None,
        }
    }

    pub fn with_details(mut self, details: impl Into<String>) -> Self {
        self.details = Some(details.into());
        self
    }
}
