//! SFU (Selective Forwarding Unit) implementation for WebRTC media routing

use crate::models::*;
use chrono::Utc;
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use uuid::Uuid;

/// SFU Router manages media rooms and participants
pub struct SfuRouter {
    rooms: Arc<RwLock<HashMap<Uuid, SfuRoom>>>,
    config: SfuConfig,
}

/// Configuration for the SFU
#[derive(Debug, Clone)]
pub struct SfuConfig {
    pub max_rooms: usize,
    pub max_participants_per_room: usize,
    pub video_codecs: Vec<String>,
    pub audio_codecs: Vec<String>,
    pub enable_simulcast: bool,
    pub enable_svc: bool,
}

impl Default for SfuConfig {
    fn default() -> Self {
        Self {
            max_rooms: 1000,
            max_participants_per_room: 20,
            video_codecs: vec!["VP8".to_string(), "VP9".to_string(), "H264".to_string()],
            audio_codecs: vec!["opus".to_string()],
            enable_simulcast: true,
            enable_svc: false,
        }
    }
}

/// SFU Room containing participants and their media tracks
pub struct SfuRoom {
    pub id: Uuid,
    pub meeting_id: Uuid,
    pub participants: HashMap<Uuid, SfuParticipant>,
    pub created_at: chrono::DateTime<Utc>,
    pub recording_enabled: bool,
}

/// SFU Participant with transports and producers/consumers
pub struct SfuParticipant {
    pub id: Uuid,
    pub user_id: Uuid,
    pub display_name: String,
    pub send_transport: Option<SfuTransport>,
    pub recv_transport: Option<SfuTransport>,
    pub producers: HashMap<String, SfuProducer>,
    pub consumers: HashMap<String, SfuConsumer>,
    pub video_quality: VideoQuality,
    pub joined_at: chrono::DateTime<Utc>,
}

/// SFU Transport for WebRTC connection
pub struct SfuTransport {
    pub id: String,
    pub direction: TransportDirection,
    pub ice_parameters: IceParameters,
    pub ice_candidates: Vec<IceCandidate>,
    pub dtls_parameters: DtlsParameters,
    pub connected: bool,
}

/// SFU Producer for sending media
pub struct SfuProducer {
    pub id: String,
    pub kind: MediaTrackKind,
    pub rtp_parameters: RtpParameters,
    pub paused: bool,
}

/// SFU Consumer for receiving media
pub struct SfuConsumer {
    pub id: String,
    pub producer_id: String,
    pub kind: MediaTrackKind,
    pub rtp_parameters: RtpParameters,
    pub paused: bool,
}

impl SfuRouter {
    /// Create a new SFU router
    pub fn new(config: SfuConfig) -> Self {
        Self {
            rooms: Arc::new(RwLock::new(HashMap::new())),
            config,
        }
    }

    /// Create a new media room
    pub async fn create_room(&self, meeting_id: Uuid) -> Result<Uuid, SfuError> {
        let mut rooms = self.rooms.write().await;
        
        if rooms.len() >= self.config.max_rooms {
            return Err(SfuError::MaxRoomsReached);
        }

        let room_id = Uuid::new_v4();
        let room = SfuRoom {
            id: room_id,
            meeting_id,
            participants: HashMap::new(),
            created_at: Utc::now(),
            recording_enabled: false,
        };

        rooms.insert(room_id, room);
        tracing::info!("Created media room: {}", room_id);
        
        Ok(room_id)
    }

    /// Get a room by ID
    pub async fn get_room(&self, room_id: Uuid) -> Option<MediaRoom> {
        let rooms = self.rooms.read().await;
        rooms.get(&room_id).map(|room| MediaRoom {
            id: room.id,
            meeting_id: room.meeting_id,
            participants: room.participants.values().map(|p| MediaParticipant {
                id: p.id,
                user_id: p.user_id,
                display_name: p.display_name.clone(),
                audio_track: p.producers.values()
                    .find(|prod| prod.kind == MediaTrackKind::Audio)
                    .map(|prod| MediaTrack {
                        id: prod.id.clone(),
                        kind: MediaTrackKind::Audio,
                        enabled: !prod.paused,
                        muted: prod.paused,
                        label: None,
                    }),
                video_track: p.producers.values()
                    .find(|prod| prod.kind == MediaTrackKind::Video)
                    .map(|prod| MediaTrack {
                        id: prod.id.clone(),
                        kind: MediaTrackKind::Video,
                        enabled: !prod.paused,
                        muted: prod.paused,
                        label: None,
                    }),
                screen_track: None,
                video_quality: p.video_quality,
                joined_at: p.joined_at,
            }).collect(),
            max_participants: self.config.max_participants_per_room,
            created_at: room.created_at,
            recording_enabled: room.recording_enabled,
        })
    }

    /// Close a room
    pub async fn close_room(&self, room_id: Uuid) -> Result<(), SfuError> {
        let mut rooms = self.rooms.write().await;
        rooms.remove(&room_id)
            .ok_or(SfuError::RoomNotFound)?;
        
        tracing::info!("Closed media room: {}", room_id);
        Ok(())
    }


    /// Add a participant to a room
    pub async fn join_room(
        &self,
        room_id: Uuid,
        user_id: Uuid,
        display_name: String,
    ) -> Result<Uuid, SfuError> {
        let mut rooms = self.rooms.write().await;
        let room = rooms.get_mut(&room_id)
            .ok_or(SfuError::RoomNotFound)?;

        if room.participants.len() >= self.config.max_participants_per_room {
            return Err(SfuError::RoomFull);
        }

        let participant_id = Uuid::new_v4();
        let participant = SfuParticipant {
            id: participant_id,
            user_id,
            display_name,
            send_transport: None,
            recv_transport: None,
            producers: HashMap::new(),
            consumers: HashMap::new(),
            video_quality: VideoQuality::default(),
            joined_at: Utc::now(),
        };

        room.participants.insert(participant_id, participant);
        tracing::info!("Participant {} joined room {}", participant_id, room_id);
        
        Ok(participant_id)
    }

    /// Remove a participant from a room
    pub async fn leave_room(&self, room_id: Uuid, participant_id: Uuid) -> Result<(), SfuError> {
        let mut rooms = self.rooms.write().await;
        let room = rooms.get_mut(&room_id)
            .ok_or(SfuError::RoomNotFound)?;

        room.participants.remove(&participant_id)
            .ok_or(SfuError::ParticipantNotFound)?;

        tracing::info!("Participant {} left room {}", participant_id, room_id);
        
        // Close room if empty
        if room.participants.is_empty() {
            drop(rooms);
            let _ = self.close_room(room_id).await;
        }

        Ok(())
    }

    /// Create a WebRTC transport for a participant
    pub async fn create_transport(
        &self,
        room_id: Uuid,
        participant_id: Uuid,
        direction: TransportDirection,
    ) -> Result<CreateTransportResponse, SfuError> {
        let mut rooms = self.rooms.write().await;
        let room = rooms.get_mut(&room_id)
            .ok_or(SfuError::RoomNotFound)?;
        
        let participant = room.participants.get_mut(&participant_id)
            .ok_or(SfuError::ParticipantNotFound)?;

        let transport_id = Uuid::new_v4().to_string();
        
        // Generate ICE parameters (in production, these would come from the WebRTC stack)
        let ice_parameters = IceParameters {
            username_fragment: generate_ice_ufrag(),
            password: generate_ice_pwd(),
            ice_lite: true,
        };

        let ice_candidates = vec![
            IceCandidate {
                foundation: "udpcandidate".to_string(),
                priority: 1078862079,
                ip: "0.0.0.0".to_string(), // Would be actual server IP
                protocol: "udp".to_string(),
                port: 10000 + (rand::random::<u16>() % 10000),
                r#type: "host".to_string(),
            },
        ];

        let dtls_parameters = DtlsParameters {
            role: "auto".to_string(),
            fingerprints: vec![
                DtlsFingerprint {
                    algorithm: "sha-256".to_string(),
                    value: generate_fingerprint(),
                },
            ],
        };

        let transport = SfuTransport {
            id: transport_id.clone(),
            direction,
            ice_parameters: ice_parameters.clone(),
            ice_candidates: ice_candidates.clone(),
            dtls_parameters: dtls_parameters.clone(),
            connected: false,
        };

        match direction {
            TransportDirection::Send => participant.send_transport = Some(transport),
            TransportDirection::Recv => participant.recv_transport = Some(transport),
        }

        tracing::info!(
            "Created {:?} transport {} for participant {}",
            direction, transport_id, participant_id
        );

        Ok(CreateTransportResponse {
            transport_id,
            ice_parameters,
            ice_candidates,
            dtls_parameters,
        })
    }

    /// Connect a transport (complete DTLS handshake)
    pub async fn connect_transport(
        &self,
        room_id: Uuid,
        participant_id: Uuid,
        transport_id: &str,
        _dtls_parameters: DtlsParameters,
    ) -> Result<(), SfuError> {
        let mut rooms = self.rooms.write().await;
        let room = rooms.get_mut(&room_id)
            .ok_or(SfuError::RoomNotFound)?;
        
        let participant = room.participants.get_mut(&participant_id)
            .ok_or(SfuError::ParticipantNotFound)?;

        // Find and connect the transport
        let transport = if let Some(ref mut t) = participant.send_transport {
            if t.id == transport_id { Some(t) } else { None }
        } else {
            None
        }.or_else(|| {
            if let Some(ref mut t) = participant.recv_transport {
                if t.id == transport_id { Some(t) } else { None }
            } else {
                None
            }
        }).ok_or(SfuError::TransportNotFound)?;

        transport.connected = true;
        tracing::info!("Connected transport {} for participant {}", transport_id, participant_id);

        Ok(())
    }

    /// Create a producer (start sending media)
    pub async fn produce(
        &self,
        room_id: Uuid,
        participant_id: Uuid,
        transport_id: &str,
        kind: MediaTrackKind,
        rtp_parameters: RtpParameters,
    ) -> Result<ProduceResponse, SfuError> {
        let mut rooms = self.rooms.write().await;
        let room = rooms.get_mut(&room_id)
            .ok_or(SfuError::RoomNotFound)?;
        
        let participant = room.participants.get_mut(&participant_id)
            .ok_or(SfuError::ParticipantNotFound)?;

        // Verify transport exists and is connected
        let transport = participant.send_transport.as_ref()
            .filter(|t| t.id == transport_id && t.connected)
            .ok_or(SfuError::TransportNotConnected)?;

        let producer_id = Uuid::new_v4().to_string();
        let producer = SfuProducer {
            id: producer_id.clone(),
            kind,
            rtp_parameters,
            paused: false,
        };

        participant.producers.insert(producer_id.clone(), producer);
        tracing::info!(
            "Created {:?} producer {} for participant {}",
            kind, producer_id, participant_id
        );

        Ok(ProduceResponse { producer_id })
    }

    /// Create a consumer (start receiving media from a producer)
    pub async fn consume(
        &self,
        room_id: Uuid,
        participant_id: Uuid,
        transport_id: &str,
        producer_id: &str,
        _rtp_capabilities: RtpCapabilities,
    ) -> Result<ConsumeResponse, SfuError> {
        let mut rooms = self.rooms.write().await;
        let room = rooms.get_mut(&room_id)
            .ok_or(SfuError::RoomNotFound)?;

        // Find the producer
        let (producer_kind, producer_rtp) = room.participants.values()
            .flat_map(|p| p.producers.values())
            .find(|p| p.id == producer_id)
            .map(|p| (p.kind, p.rtp_parameters.clone()))
            .ok_or(SfuError::ProducerNotFound)?;

        let participant = room.participants.get_mut(&participant_id)
            .ok_or(SfuError::ParticipantNotFound)?;

        // Verify transport exists and is connected
        let _transport = participant.recv_transport.as_ref()
            .filter(|t| t.id == transport_id && t.connected)
            .ok_or(SfuError::TransportNotConnected)?;

        let consumer_id = Uuid::new_v4().to_string();
        let consumer = SfuConsumer {
            id: consumer_id.clone(),
            producer_id: producer_id.to_string(),
            kind: producer_kind,
            rtp_parameters: producer_rtp.clone(),
            paused: false,
        };

        participant.consumers.insert(consumer_id.clone(), consumer);
        tracing::info!(
            "Created consumer {} for producer {} for participant {}",
            consumer_id, producer_id, participant_id
        );

        Ok(ConsumeResponse {
            consumer_id,
            producer_id: producer_id.to_string(),
            kind: producer_kind,
            rtp_parameters: producer_rtp,
        })
    }

    /// Pause a producer
    pub async fn pause_producer(
        &self,
        room_id: Uuid,
        participant_id: Uuid,
        producer_id: &str,
    ) -> Result<(), SfuError> {
        let mut rooms = self.rooms.write().await;
        let room = rooms.get_mut(&room_id)
            .ok_or(SfuError::RoomNotFound)?;
        
        let participant = room.participants.get_mut(&participant_id)
            .ok_or(SfuError::ParticipantNotFound)?;

        let producer = participant.producers.get_mut(producer_id)
            .ok_or(SfuError::ProducerNotFound)?;

        producer.paused = true;
        tracing::info!("Paused producer {} for participant {}", producer_id, participant_id);

        Ok(())
    }

    /// Resume a producer
    pub async fn resume_producer(
        &self,
        room_id: Uuid,
        participant_id: Uuid,
        producer_id: &str,
    ) -> Result<(), SfuError> {
        let mut rooms = self.rooms.write().await;
        let room = rooms.get_mut(&room_id)
            .ok_or(SfuError::RoomNotFound)?;
        
        let participant = room.participants.get_mut(&participant_id)
            .ok_or(SfuError::ParticipantNotFound)?;

        let producer = participant.producers.get_mut(producer_id)
            .ok_or(SfuError::ProducerNotFound)?;

        producer.paused = false;
        tracing::info!("Resumed producer {} for participant {}", producer_id, participant_id);

        Ok(())
    }

    /// Set video quality for a participant
    pub async fn set_video_quality(
        &self,
        room_id: Uuid,
        participant_id: Uuid,
        quality: VideoQuality,
    ) -> Result<(), SfuError> {
        let mut rooms = self.rooms.write().await;
        let room = rooms.get_mut(&room_id)
            .ok_or(SfuError::RoomNotFound)?;
        
        let participant = room.participants.get_mut(&participant_id)
            .ok_or(SfuError::ParticipantNotFound)?;

        let previous_quality = participant.video_quality;
        participant.video_quality = quality;

        tracing::info!(
            "Changed video quality for participant {} from {:?} to {:?}",
            participant_id, previous_quality, quality
        );

        Ok(())
    }

    /// Get RTP capabilities for the router
    pub fn get_rtp_capabilities(&self) -> RtpCapabilities {
        RtpCapabilities {
            codecs: vec![
                RtpCodec {
                    mime_type: "audio/opus".to_string(),
                    payload_type: 111,
                    clock_rate: 48000,
                    channels: Some(2),
                    parameters: None,
                },
                RtpCodec {
                    mime_type: "video/VP8".to_string(),
                    payload_type: 96,
                    clock_rate: 90000,
                    channels: None,
                    parameters: None,
                },
                RtpCodec {
                    mime_type: "video/H264".to_string(),
                    payload_type: 102,
                    clock_rate: 90000,
                    channels: None,
                    parameters: Some(serde_json::json!({
                        "level-asymmetry-allowed": 1,
                        "packetization-mode": 1,
                        "profile-level-id": "42e01f"
                    })),
                },
            ],
            header_extensions: vec![
                RtpHeaderExtension {
                    uri: "urn:ietf:params:rtp-hdrext:sdes:mid".to_string(),
                    id: 1,
                    encrypt: false,
                },
                RtpHeaderExtension {
                    uri: "urn:ietf:params:rtp-hdrext:ssrc-audio-level".to_string(),
                    id: 10,
                    encrypt: false,
                },
            ],
        }
    }
}

/// SFU errors
#[derive(Debug, thiserror::Error)]
pub enum SfuError {
    #[error("Maximum number of rooms reached")]
    MaxRoomsReached,
    #[error("Room not found")]
    RoomNotFound,
    #[error("Room is full")]
    RoomFull,
    #[error("Participant not found")]
    ParticipantNotFound,
    #[error("Transport not found")]
    TransportNotFound,
    #[error("Transport not connected")]
    TransportNotConnected,
    #[error("Producer not found")]
    ProducerNotFound,
    #[error("Consumer not found")]
    ConsumerNotFound,
}

// Helper functions for generating WebRTC parameters
fn generate_ice_ufrag() -> String {
    use rand::Rng;
    let mut rng = rand::thread_rng();
    (0..8).map(|_| rng.sample(rand::distributions::Alphanumeric) as char).collect()
}

fn generate_ice_pwd() -> String {
    use rand::Rng;
    let mut rng = rand::thread_rng();
    (0..24).map(|_| rng.sample(rand::distributions::Alphanumeric) as char).collect()
}

fn generate_fingerprint() -> String {
    use rand::Rng;
    let mut rng = rand::thread_rng();
    let bytes: Vec<String> = (0..32).map(|_| format!("{:02X}", rng.gen::<u8>())).collect();
    bytes.join(":")
}
