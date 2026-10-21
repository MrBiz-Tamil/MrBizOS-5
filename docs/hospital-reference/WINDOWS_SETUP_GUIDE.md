# Windows Setup Guide for MediConsult Hub Backend

## Issue Fixed

**Error:** `FastAPIError: Invalid args for response field! Hint: check that <class 'app.models.models.ChatRoom'> is a valid Pydantic field type`

**Root Cause:** Missing Pydantic schemas for ChatRoom and ChatMessage in `app/schemas/schemas.py`. The endpoint was trying to use SQLAlchemy models directly as FastAPI response models.

**Solution:** Added proper Pydantic schemas for all chat-related models.

---

## Prerequisites

1. **Python 3.8+** installed
2. **pip** package manager
3. **Virtual environment** (recommended)

---

## Step-by-Step Setup Instructions

### 1. Navigate to Backend Directory

```bash
cd "C:\Users\balar\Downloads\Cambiar Backend FastAPI\workspace\uploads\medi-consultation-Doctors-app\medi-consultation-Doctors-app\backend"
```

### 2. Create and Activate Virtual Environment

```bash
# Create virtual environment
python -m venv venv

# Activate virtual environment
venv\Scripts\activate
```

### 3. Install Dependencies

```bash
pip install -r requirements.txt
```

### 4. Initialize Database

```bash
python init_db.py
```

**Expected Output:**
```
Creating database tables...
✓ Database tables created successfully!

Created tables:
- users
- doctors
- clinics
- patients
...
```

### 5. Seed Sample Data

```bash
python seed_data.py
```

**Expected Output:**
```
🌱 Seeding database with sample data...
📍 Creating clinics...
✓ Created 3 clinics
👨‍⚕️ Creating doctors...
✓ Created 5 doctors
...
✅ Database seeded successfully!
```

### 6. Start the Server

```bash
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
```

**Expected Output:**
```
INFO:     Uvicorn running on http://0.0.0.0:8000 (Press CTRL+C to quit)
INFO:     Started reloader process [xxxxx] using WatchFiles
INFO:     Started server process [xxxxx]
INFO:     Waiting for application startup.
INFO:     Application startup complete.
```

### 7. Verify Installation

Open your browser and navigate to:
- **API Documentation:** http://localhost:8000/docs
- **Health Check:** http://localhost:8000/health

---

## Test Credentials

### Patient Account
- **Email:** `john.smith@email.com`
- **Password:** `patient123`

### Doctor Account
- **Email:** `dr.sarah@mediconsult.com`
- **Password:** `doctor123`

---

## Testing the API

### 1. Health Check

```bash
curl http://localhost:8000/health
```

**Expected Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "environment": "development"
}
```

### 2. Login

```bash
curl -X POST "http://localhost:8000/api/v1/auth/login" ^
  -H "Content-Type: application/x-www-form-urlencoded" ^
  -d "username=john.smith@email.com&password=patient123"
```

**Expected Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer"
}
```

### 3. Get Doctors List

```bash
# First, get the token
set TOKEN=<your_token_from_login>

# Then make the request
curl -X GET "http://localhost:8000/api/v1/doctors/" ^
  -H "Authorization: Bearer %TOKEN%"
```

---

## Common Issues and Solutions

### Issue 1: Port Already in Use

**Error:** `OSError: [WinError 10048] Only one usage of each socket address`

**Solution:**
```bash
# Find the process using port 8000
netstat -ano | findstr :8000

# Kill the process (replace PID with actual process ID)
taskkill /PID <PID> /F

# Or use a different port
uvicorn app.main:app --reload --host 0.0.0.0 --port 8001
```

### Issue 2: Module Not Found

**Error:** `ModuleNotFoundError: No module named 'fastapi'`

**Solution:**
```bash
# Make sure virtual environment is activated
venv\Scripts\activate

# Reinstall dependencies
pip install -r requirements.txt
```

### Issue 3: Database Locked

**Error:** `sqlite3.OperationalError: database is locked`

**Solution:**
```bash
# Stop the server (Ctrl+C)
# Delete the database
del mediconsult.db

# Reinitialize
python init_db.py
python seed_data.py
```

### Issue 4: Import Error with Pydantic

**Error:** `ImportError: cannot import name 'ChatRoom' from 'app.schemas.schemas'`

**Solution:**
This has been fixed in the latest version. Make sure you have the updated `app/schemas/schemas.py` file with ChatRoom and ChatMessage schemas.

---

## Project Structure

```
backend/
├── app/
│   ├── api/
│   │   ├── endpoints/
│   │   │   ├── auth.py
│   │   │   ├── doctors.py
│   │   │   ├── patients.py
│   │   │   ├── appointments.py
│   │   │   ├── chat.py          # Fixed: Now uses proper Pydantic schemas
│   │   │   └── ...
│   │   └── api.py
│   ├── core/
│   │   └── config.py
│   ├── crud/
│   │   └── crud.py
│   ├── db/
│   │   └── base.py
│   ├── models/
│   │   └── models.py
│   ├── schemas/
│   │   └── schemas.py           # Fixed: Added ChatRoom and ChatMessage schemas
│   └── main.py
├── tests/
├── init_db.py
├── seed_data.py
├── requirements.txt
└── mediconsult.db
```

---

## Features Implemented

### ✅ Core Features
- User Authentication (JWT)
- Doctor Management
- Patient Management
- Appointment Scheduling
- Queue Management
- Review & Rating System
- Electronic Health Records (EHR)
- Emergency Services
- Consultations
- Prescriptions
- Chat System (Fixed)
- Video Sessions

### ✅ Security Features
- Password validation (8+ chars, uppercase, digit)
- JWT token authentication
- Input validation
- SQL injection prevention
- CORS configuration
- Connection pooling

---

## API Endpoints

### Authentication
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/register` - User registration

### Doctors
- `GET /api/v1/doctors/` - List all doctors
- `GET /api/v1/doctors/{id}` - Get doctor by ID
- `POST /api/v1/doctors/` - Create doctor
- `PUT /api/v1/doctors/{id}` - Update doctor

### Patients
- `GET /api/v1/patients/` - List all patients
- `GET /api/v1/patients/{id}` - Get patient by ID
- `POST /api/v1/patients/` - Create patient
- `PUT /api/v1/patients/{id}` - Update patient

### Appointments
- `GET /api/v1/appointments/patient/{id}` - Get patient appointments
- `GET /api/v1/appointments/doctor/{id}` - Get doctor appointments
- `POST /api/v1/appointments/` - Create appointment
- `PUT /api/v1/appointments/{id}` - Update appointment

### Queue Management
- `GET /api/v1/queue/doctor/{id}/today` - Get today's queue
- `POST /api/v1/queue/appointment/{id}/check-in` - Check-in
- `POST /api/v1/queue/appointment/{id}/call-next` - Call next patient

### Reviews
- `GET /api/v1/reviews/doctor/{id}` - Get doctor reviews
- `GET /api/v1/reviews/doctor/{id}/stats` - Get review statistics
- `POST /api/v1/reviews/` - Submit review

### Chat (Fixed)
- `POST /api/v1/chat/rooms` - Create chat room
- `GET /api/v1/chat/rooms/user/{id}` - Get user's chat rooms
- `GET /api/v1/chat/messages/{room_id}` - Get chat messages
- `POST /api/v1/chat/messages` - Send message

### Emergency
- `GET /api/v1/emergency/contacts/patient/{id}` - Get emergency contacts
- `POST /api/v1/emergency/sos` - Trigger SOS
- `GET /api/v1/emergency/nearby-hospitals` - Find nearby hospitals

---

## Environment Variables

Create a `.env` file in the backend directory:

```env
# Security
SECRET_KEY=your-super-secret-key-change-this-in-production
ENVIRONMENT=development

# Database
DATABASE_URL=sqlite:///./mediconsult.db

# CORS
BACKEND_CORS_ORIGINS=http://localhost:3000,http://localhost:8000

# Notification Services (Optional)
SENDGRID_API_KEY=your-sendgrid-api-key
TWILIO_ACCOUNT_SID=your-twilio-account-sid
TWILIO_AUTH_TOKEN=your-twilio-auth-token
TWILIO_PHONE_NUMBER=your-twilio-phone-number
```

---

## Development Tips

### 1. Auto-reload on Changes

The `--reload` flag enables automatic reloading when code changes:
```bash
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
```

### 2. View Logs

Logs are printed to the console. For production, configure proper logging:
```python
import logging
logging.basicConfig(level=logging.INFO)
```

### 3. Database Migrations

For schema changes:
```bash
# Delete old database
del mediconsult.db

# Reinitialize
python init_db.py
python seed_data.py
```

### 4. Testing

Run tests with pytest:
```bash
pytest tests/ -v
```

---

## Production Deployment

### 1. Update Environment Variables

```env
ENVIRONMENT=production
SECRET_KEY=<generate-secure-key>
DATABASE_URL=postgresql://user:pass@host/db
```

### 2. Use Production Server

```bash
# Install gunicorn
pip install gunicorn

# Run with gunicorn
gunicorn app.main:app -w 4 -k uvicorn.workers.UvicornWorker --bind 0.0.0.0:8000
```

### 3. Enable HTTPS

Use a reverse proxy like Nginx with SSL certificate.

---

## Support

For issues or questions:
1. Check the API documentation at `/docs`
2. Review the test results in `TEST_RESULTS.md`
3. Check security fixes in `SECURITY_FIXES.md`

---

## Changelog

### Version 1.0.1 (2024-01-15)
- **Fixed:** Added missing Pydantic schemas for ChatRoom and ChatMessage
- **Fixed:** EmergencyContact model naming conflict (relationship → relationship_type)
- **Added:** Comprehensive Windows setup guide
- **Added:** Test credentials and sample data
- **Improved:** Error handling and validation

### Version 1.0.0 (2024-01-10)
- Initial release
- Core features implemented
- Security features added
- Sample data generator

---

**Last Updated:** 2024-01-15
**Version:** 1.0.1