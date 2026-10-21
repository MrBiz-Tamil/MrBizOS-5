# Video Consultation with WebRTC - Implementation Guide

This guide explains how to implement video consultation functionality using WebRTC in the MediConsult Hub application.

## Backend Architecture

### WebSocket Signaling Server

The backend provides a WebSocket endpoint for WebRTC signaling at:
```
ws://localhost:8000/api/v1/video/ws/{room_id}/{user_id}
```

### Key Features

1. **Real-time Signaling**: Handles WebRTC offer, answer, and ICE candidate exchange
2. **Room Management**: Manages participants in video consultation rooms
3. **Session Tracking**: Stores video session metadata in the database
4. **Connection Management**: Handles user connections, disconnections, and notifications

## Database Schema

### VideoSession Table
```python
- id: Unique session identifier
- appointment_id: Link to appointment
- room_id: Unique room identifier for WebRTC
- start_time: Session start timestamp
- end_time: Session end timestamp (nullable)
- duration: Session duration in seconds
- status: active, ended, cancelled
- recording_url: URL to recorded session (optional)
```

## API Endpoints

### REST Endpoints

1. **Create Video Session**
   ```
   POST /api/v1/video/sessions
   Body: {
     "appointment_id": "string",
     "room_id": "string"
   }
   ```

2. **Get Video Session**
   ```
   GET /api/v1/video/sessions/{session_id}
   ```

3. **Get Session by Appointment**
   ```
   GET /api/v1/video/sessions/appointment/{appointment_id}
   ```

4. **Update Video Session**
   ```
   PUT /api/v1/video/sessions/{session_id}
   Body: {
     "end_time": "datetime",
     "status": "ended",
     "duration": 1800
   }
   ```

5. **Get Active Participants**
   ```
   GET /api/v1/video/sessions/{session_id}/participants
   ```

### WebSocket Messages

#### Client to Server

1. **Offer** (Initiating call)
   ```json
   {
     "type": "offer",
     "offer": {
       "sdp": "...",
       "type": "offer"
     },
     "target_user": "user_id"
   }
   ```

2. **Answer** (Accepting call)
   ```json
   {
     "type": "answer",
     "answer": {
       "sdp": "...",
       "type": "answer"
     },
     "target_user": "user_id"
   }
   ```

3. **ICE Candidate**
   ```json
   {
     "type": "ice_candidate",
     "candidate": {
       "candidate": "...",
       "sdpMLineIndex": 0,
       "sdpMid": "0"
     },
     "target_user": "user_id"
   }
   ```

4. **Leave Call**
   ```json
   {
     "type": "leave"
   }
   ```

5. **Chat Message**
   ```json
   {
     "type": "chat_message",
     "message": "Hello!"
   }
   ```

#### Server to Client

1. **User Joined**
   ```json
   {
     "type": "user_joined",
     "user_id": "user_id",
     "timestamp": "2024-01-01T00:00:00"
   }
   ```

2. **Participants List**
   ```json
   {
     "type": "participants_list",
     "participants": ["user_id_1", "user_id_2"]
   }
   ```

3. **Offer Received**
   ```json
   {
     "type": "offer",
     "offer": {...},
     "from_user": "user_id"
   }
   ```

4. **Answer Received**
   ```json
   {
     "type": "answer",
     "answer": {...},
     "from_user": "user_id"
   }
   ```

5. **ICE Candidate Received**
   ```json
   {
     "type": "ice_candidate",
     "candidate": {...},
     "from_user": "user_id"
   }
   ```

6. **User Left**
   ```json
   {
     "type": "user_left",
     "user_id": "user_id",
     "timestamp": "2024-01-01T00:00:00"
   }
   ```

7. **User Disconnected**
   ```json
   {
     "type": "user_disconnected",
     "user_id": "user_id",
     "timestamp": "2024-01-01T00:00:00"
   }
   ```

## Frontend Implementation Guide

### 1. Install Required Dependencies

```bash
npm install simple-peer
```

### 2. WebSocket Connection

```javascript
const ws = new WebSocket(`ws://localhost:8000/api/v1/video/ws/${roomId}/${userId}`);

ws.onopen = () => {
  console.log('Connected to signaling server');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  handleSignalingMessage(message);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Disconnected from signaling server');
};
```

### 3. WebRTC Peer Connection

```javascript
import SimplePeer from 'simple-peer';

// Create peer connection
const peer = new SimplePeer({
  initiator: isInitiator, // true for caller, false for receiver
  trickle: true,
  stream: localStream // Your local media stream
});

// Handle signaling data
peer.on('signal', (data) => {
  // Send offer/answer/ice-candidate through WebSocket
  ws.send(JSON.stringify({
    type: data.type === 'offer' ? 'offer' : 
          data.type === 'answer' ? 'answer' : 'ice_candidate',
    [data.type === 'offer' ? 'offer' : 
     data.type === 'answer' ? 'answer' : 'candidate']: data,
    target_user: targetUserId
  }));
});

// Handle incoming stream
peer.on('stream', (remoteStream) => {
  // Display remote video
  remoteVideoElement.srcObject = remoteStream;
});

// Handle connection
peer.on('connect', () => {
  console.log('Peer connected');
});

// Handle errors
peer.on('error', (err) => {
  console.error('Peer error:', err);
});
```

### 4. Handle Signaling Messages

```javascript
function handleSignalingMessage(message) {
  switch (message.type) {
    case 'participants_list':
      // Initialize calls with existing participants
      message.participants.forEach(participantId => {
        initiateCall(participantId);
      });
      break;
      
    case 'user_joined':
      // New user joined, initiate call
      initiateCall(message.user_id);
      break;
      
    case 'offer':
      // Received offer, create answer
      handleOffer(message.offer, message.from_user);
      break;
      
    case 'answer':
      // Received answer, complete connection
      handleAnswer(message.answer, message.from_user);
      break;
      
    case 'ice_candidate':
      // Add ICE candidate
      handleIceCandidate(message.candidate, message.from_user);
      break;
      
    case 'user_left':
    case 'user_disconnected':
      // Remove user's video
      removeUserVideo(message.user_id);
      break;
      
    case 'chat_message':
      // Display chat message
      displayChatMessage(message.message, message.from_user);
      break;
  }
}
```

### 5. Get Local Media Stream

```javascript
async function getLocalStream() {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({
      video: {
        width: { ideal: 1280 },
        height: { ideal: 720 }
      },
      audio: {
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true
      }
    });
    
    localVideoElement.srcObject = stream;
    return stream;
  } catch (error) {
    console.error('Error accessing media devices:', error);
    throw error;
  }
}
```

### 6. Complete React Component Example

```jsx
import React, { useEffect, useRef, useState } from 'react';
import SimplePeer from 'simple-peer';

function VideoConsultation({ roomId, userId, appointmentId }) {
  const [peers, setPeers] = useState({});
  const [localStream, setLocalStream] = useState(null);
  const localVideoRef = useRef();
  const wsRef = useRef();

  useEffect(() => {
    initializeVideoCall();
    return () => cleanup();
  }, []);

  async function initializeVideoCall() {
    // Get local stream
    const stream = await navigator.mediaDevices.getUserMedia({
      video: true,
      audio: true
    });
    
    setLocalStream(stream);
    localVideoRef.current.srcObject = stream;

    // Connect to WebSocket
    const ws = new WebSocket(
      `ws://localhost:8000/api/v1/video/ws/${roomId}/${userId}`
    );
    
    wsRef.current = ws;

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      handleSignalingMessage(message, stream);
    };
  }

  function handleSignalingMessage(message, stream) {
    switch (message.type) {
      case 'participants_list':
        message.participants.forEach(participantId => {
          createPeer(participantId, true, stream);
        });
        break;
        
      case 'user_joined':
        createPeer(message.user_id, true, stream);
        break;
        
      case 'offer':
        createPeer(message.from_user, false, stream, message.offer);
        break;
        
      case 'answer':
        if (peers[message.from_user]) {
          peers[message.from_user].signal(message.answer);
        }
        break;
        
      case 'ice_candidate':
        if (peers[message.from_user]) {
          peers[message.from_user].signal(message.candidate);
        }
        break;
        
      case 'user_left':
      case 'user_disconnected':
        removePeer(message.user_id);
        break;
    }
  }

  function createPeer(peerId, initiator, stream, offerSignal = null) {
    const peer = new SimplePeer({
      initiator,
      trickle: true,
      stream
    });

    peer.on('signal', (signal) => {
      wsRef.current.send(JSON.stringify({
        type: signal.type === 'offer' ? 'offer' : 
              signal.type === 'answer' ? 'answer' : 'ice_candidate',
        [signal.type === 'offer' ? 'offer' : 
         signal.type === 'answer' ? 'answer' : 'candidate']: signal,
        target_user: peerId
      }));
    });

    peer.on('stream', (remoteStream) => {
      setPeers(prev => ({
        ...prev,
        [peerId]: { peer, stream: remoteStream }
      }));
    });

    if (offerSignal) {
      peer.signal(offerSignal);
    }

    setPeers(prev => ({
      ...prev,
      [peerId]: { peer, stream: null }
    }));
  }

  function removePeer(peerId) {
    if (peers[peerId]) {
      peers[peerId].peer.destroy();
      setPeers(prev => {
        const newPeers = { ...prev };
        delete newPeers[peerId];
        return newPeers;
      });
    }
  }

  function cleanup() {
    // Close all peer connections
    Object.values(peers).forEach(({ peer }) => peer.destroy());
    
    // Stop local stream
    if (localStream) {
      localStream.getTracks().forEach(track => track.stop());
    }
    
    // Close WebSocket
    if (wsRef.current) {
      wsRef.current.send(JSON.stringify({ type: 'leave' }));
      wsRef.current.close();
    }
  }

  return (
    <div className="video-consultation">
      <div className="local-video">
        <video ref={localVideoRef} autoPlay muted />
      </div>
      
      <div className="remote-videos">
        {Object.entries(peers).map(([peerId, { stream }]) => (
          stream && (
            <RemoteVideo key={peerId} stream={stream} peerId={peerId} />
          )
        ))}
      </div>
      
      <div className="controls">
        <button onClick={() => {
          localStream.getAudioTracks()[0].enabled = 
            !localStream.getAudioTracks()[0].enabled;
        }}>
          Toggle Mic
        </button>
        
        <button onClick={() => {
          localStream.getVideoTracks()[0].enabled = 
            !localStream.getVideoTracks()[0].enabled;
        }}>
          Toggle Video
        </button>
        
        <button onClick={cleanup}>
          End Call
        </button>
      </div>
    </div>
  );
}

function RemoteVideo({ stream, peerId }) {
  const videoRef = useRef();
  
  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream;
    }
  }, [stream]);
  
  return (
    <div className="remote-video">
      <video ref={videoRef} autoPlay />
      <span>{peerId}</span>
    </div>
  );
}

export default VideoConsultation;
```

## Testing

1. **Start the backend server**:
   ```bash
   cd backend
   uvicorn app.main:app --reload
   ```

2. **Test WebSocket connection**:
   - Use a WebSocket client (e.g., Postman, wscat)
   - Connect to: `ws://localhost:8000/api/v1/video/ws/test-room/user1`

3. **Test with multiple users**:
   - Open multiple browser tabs
   - Connect each with different user IDs
   - Verify signaling messages are exchanged

## Security Considerations

1. **Authentication**: All endpoints require JWT authentication
2. **Room Access**: Verify users have permission to join specific rooms
3. **STUN/TURN Servers**: Configure for production NAT traversal
4. **Encryption**: WebRTC uses DTLS-SRTP for media encryption
5. **Rate Limiting**: Implement rate limiting on WebSocket connections

## Production Deployment

1. **Use TURN servers** for NAT traversal:
   ```javascript
   const peer = new SimplePeer({
     initiator: true,
     trickle: true,
     stream: localStream,
     config: {
       iceServers: [
         { urls: 'stun:stun.l.google.com:19302' },
         {
           urls: 'turn:your-turn-server.com:3478',
           username: 'username',
           credential: 'password'
         }
       ]
     }
   });
   ```

2. **Enable WSS** (WebSocket Secure) in production
3. **Implement recording** if required (server-side or client-side)
4. **Add quality monitoring** and adaptive bitrate
5. **Handle reconnection** logic for network interruptions

## Troubleshooting

1. **No video/audio**: Check browser permissions
2. **Connection fails**: Verify STUN/TURN server configuration
3. **One-way audio/video**: Check firewall/NAT settings
4. **Poor quality**: Adjust video constraints and bitrate
5. **WebSocket disconnects**: Implement reconnection logic

## Additional Features

- Screen sharing
- Recording functionality
- Virtual backgrounds
- Noise cancellation
- Bandwidth optimization
- Multi-party conferencing
- Chat during video call
- File sharing during consultation