//! HTTP handlers for the media service

use axum::{
    extract::{Path, State},
    http::StatusCode,
    Json,
};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use uuid::Uuid;

use crate::models::{JoinRoomRequest, JoinRoomResponse, QualityChangeRequest, ScreenShareConfig};
use crate::sfu::{SfuError, SfuService};

/// Application state
pub struct AppState {
    pub sfu: SfuService,
}

#[derive(Serialize)]
pub struct HealthResponse {
    pub status: String,
    pub version: String,
}

pub async fn health_check() -> Json<HealthResponse> {
    Json(HealthResponse {
        status: "healthy".to_string(),
        version: env!("CARGO_PKG_VERSION").to_string(),
    })
}

/// Error response
#[derive(Serialize)]
pub struct ErrorResponse {
    pub error: String,
    pub code: String,
}

impl From<SfuError> for (StatusCode, Json<ErrorResponse>) {
    fn from(err: SfuError) -> Self {
        let (status, code) = match &err {
            SfuError::RoomNotFound => (StatusCode::NOT_FOUND, "ROOM_NOT_FOUND"),
            SfuError::RoomFull => (StatusCode::CONFLICT, "ROOM_FULL"),
            SfuError::ParticipantNotFound => (StatusCode::NOT_FOUND, "PARTICIPANT_NOT_FOUND"),
            SfuError::ScreenShareInUse => (StatusCode::CONFLICT, "SCREEN_SHARE_IN_USE"),
            SfuError::WebRtcError(_) => (StatusCode::INTERNAL_SERVER_ERROR, "WEBRTC_ERROR"),
            SfuError::Internal(_) => (StatusCode::INTERNAL_SERVER_ERROR, "INTERNAL_ERROR"),
        };

        (
            status,
            Json(ErrorResponse {
                error: err.to_string(),
                code: code.to_string(),
            }),
        )
    }
}

/// Join a media room
pub async fn join_room(
    State(state): State<Arc<AppState>>,
    Json(req): Json<JoinRoomRequest>,
) -> Result<Json<JoinRoomResponse>, (StatusCode, Json<ErrorResponse>)> {
    let (room, participant) = state
        .sfu
        .join_room(
            req.meeting_id,
            req.user_id,
            req.display_name,
            req.audio_enabled,
            req.video_enabled,
        )
        .map_err(|e| e.into())?;

    Ok(Json(JoinRoomResponse {
        room_id: room.room.id,
        participant_id: participant.id,
        ice_servers: state.sfu.get_ice_servers(),
        participants: room.get_participants(),
    }))
}

/// Leave a media room
pub async fn leave_room(
    State(state): State<Arc<AppState>>,
    Path((meeting_id, participant_id)): Path<(Uuid, Uuid)>,
) -> Result<StatusCode, (StatusCode, Json<ErrorResponse>)> {
    state
        .sfu
        .leave_room(meeting_id, participant_id)
        .map_err(|e| e.into())?;

    Ok(StatusCode::NO_CONTENT)
}

/// Update media state request
#[derive(Deserialize)]
pub struct UpdateMediaStateRequest {
    pub audio_enabled: bool,
    pub video_enabled: bool,
}

/// Update participant media state
pub async fn update_media_state(
    State(state): State<Arc<AppState>>,
    Path((meeting_id, participant_id)): Path<(Uuid, Uuid)>,
    Json(req): Json<UpdateMediaStateRequest>,
) -> Result<StatusCode, (StatusCode, Json<ErrorResponse>)> {
    state
        .sfu
        .update_media_state(meeting_id, participant_id, req.audio_enabled, req.video_enabled)
        .map_err(|e| e.into())?;

    Ok(StatusCode::OK)
}

/// Start screen share
pub async fn start_screen_share(
    State(state): State<Arc<AppState>>,
    Path((meeting_id, participant_id)): Path<(Uuid, Uuid)>,
    Json(config): Json<Option<ScreenShareConfig>>,
) -> Result<Json<ScreenShareConfig>, (StatusCode, Json<ErrorResponse>)> {
    let config = state
        .sfu
        .start_screen_share(meeting_id, participant_id, config)
        .map_err(|e| e.into())?;

    Ok(Json(config))
}

/// Stop screen share
pub async fn stop_screen_share(
    State(state): State<Arc<AppState>>,
    Path((meeting_id, participant_id)): Path<(Uuid, Uuid)>,
) -> Result<StatusCode, (StatusCode, Json<ErrorResponse>)> {
    state
        .sfu
        .stop_screen_share(meeting_id, participant_id)
        .map_err(|e| e.into())?;

    Ok(StatusCode::OK)
}

/// Update video quality
pub async fn update_quality(
    State(state): State<Arc<AppState>>,
    Path((meeting_id, participant_id)): Path<(Uuid, Uuid)>,
    Json(req): Json<QualityChangeRequest>,
) -> Result<StatusCode, (StatusCode, Json<ErrorResponse>)> {
    state
        .sfu
        .update_quality(meeting_id, participant_id, req.quality)
        .map_err(|e| e.into())?;

    Ok(StatusCode::OK)
}

/// Bandwidth estimate request
#[derive(Deserialize)]
pub struct BandwidthEstimateRequest {
    pub available_bitrate: u32,
    pub packet_loss: f32,
    pub round_trip_time: u32,
}

/// Process bandwidth estimate
pub async fn process_bandwidth(
    State(state): State<Arc<AppState>>,
    Path((meeting_id, participant_id)): Path<(Uuid, Uuid)>,
    Json(req): Json<BandwidthEstimateRequest>,
) -> Result<Json<crate::models::BandwidthEstimate>, (StatusCode, Json<ErrorResponse>)> {
    let estimate = state
        .sfu
        .process_bandwidth_estimate(
            meeting_id,
            participant_id,
            req.available_bitrate,
            req.packet_loss,
            req.round_trip_time,
        )
        .map_err(|e| e.into())?;

    Ok(Json(estimate))
}

/// Get room statistics
pub async fn get_room_stats(
    State(state): State<Arc<AppState>>,
    Path(meeting_id): Path<Uuid>,
) -> Result<Json<crate::sfu::RoomStats>, (StatusCode, Json<ErrorResponse>)> {
    let stats = state.sfu.get_room_stats(meeting_id).map_err(|e| e.into())?;
    Ok(Json(stats))
}

/// Close a room
pub async fn close_room(
    State(state): State<Arc<AppState>>,
    Path(meeting_id): Path<Uuid>,
) -> Result<StatusCode, (StatusCode, Json<ErrorResponse>)> {
    state.sfu.close_room(meeting_id).map_err(|e| e.into())?;
    Ok(StatusCode::NO_CONTENT)
}
