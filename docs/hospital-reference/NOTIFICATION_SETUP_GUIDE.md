# Notification Setup Guide - Email & WhatsApp Integration

This guide will help you set up email and WhatsApp notifications for appointment reminders.

## Overview

The notification system supports:
- **Email** notifications via SendGrid
- **WhatsApp** messages via Twilio
- **SMS** messages via Twilio

## Prerequisites

1. **SendGrid Account** (for email)
   - Sign up at https://sendgrid.com
   - Free tier: 100 emails/day

2. **Twilio Account** (for WhatsApp and SMS)
   - Sign up at https://www.twilio.com
   - Free trial: $15 credit

## Setup Instructions

### 1. SendGrid Setup (Email)

#### Step 1: Create SendGrid Account
1. Go to https://sendgrid.com/pricing/
2. Sign up for free account (100 emails/day)
3. Verify your email address

#### Step 2: Create API Key
1. Log in to SendGrid dashboard
2. Go to Settings → API Keys
3. Click "Create API Key"
4. Name: "MediConsult Hub"
5. Permission: "Full Access"
6. Copy the API key (you won't see it again!)

#### Step 3: Verify Sender Identity
1. Go to Settings → Sender Authentication
2. Click "Verify a Single Sender"
3. Fill in your details:
   - From Name: MediConsult Hub
   - From Email: noreply@yourdomain.com (or your email)
4. Verify the email sent to you

#### Step 4: Add to Environment Variables
```bash
SENDGRID_API_KEY=SG.xxxxxxxxxxxxxxxxxxxxxxxxxxxxx
FROM_EMAIL=noreply@yourdomain.com
```

---

### 2. Twilio Setup (WhatsApp & SMS)

#### Step 1: Create Twilio Account
1. Go to https://www.twilio.com/try-twilio
2. Sign up for free trial ($15 credit)
3. Verify your phone number

#### Step 2: Get Account Credentials
1. Log in to Twilio Console
2. Copy from dashboard:
   - Account SID
   - Auth Token

#### Step 3: Set Up WhatsApp Sandbox
1. In Twilio Console, go to Messaging → Try it out → Send a WhatsApp message
2. Follow instructions to join sandbox:
   - Send "join <your-sandbox-name>" to the Twilio WhatsApp number
   - Default: +1 415 523 8886
3. Note: Sandbox is for testing. For production, you need WhatsApp Business approval

#### Step 4: Get Phone Number (for SMS)
1. Go to Phone Numbers → Manage → Buy a number
2. Select a number with SMS capability
3. Purchase (uses trial credit)

#### Step 5: Add to Environment Variables
```bash
TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TWILIO_AUTH_TOKEN=your_auth_token_here
TWILIO_PHONE_NUMBER=+1234567890
TWILIO_WHATSAPP_NUMBER=whatsapp:+14155238886
```

---

### 3. Configure Backend

#### Update .env file
```bash
cd backend
cp .env.example .env
nano .env  # or use your preferred editor
```

Add your credentials:
```env
# SendGrid
SENDGRID_API_KEY=SG.xxxxxxxxxxxxxxxxxxxxxxxxxxxxx
FROM_EMAIL=noreply@yourdomain.com

# Twilio
TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TWILIO_AUTH_TOKEN=your_auth_token_here
TWILIO_PHONE_NUMBER=+1234567890
TWILIO_WHATSAPP_NUMBER=whatsapp:+14155238886
```

#### Install Dependencies
```bash
pip install sendgrid twilio python-dotenv
```

#### Restart Backend
```bash
uvicorn app.main:app --reload
```

---

## Testing

### Test Email
```bash
curl -X POST "http://localhost:8000/api/v1/reminders/test-email?email=your@email.com"
```

### Test WhatsApp
```bash
curl -X POST "http://localhost:8000/api/v1/reminders/test-whatsapp?phone=+1234567890"
```

### Test Full Reminder System
1. Create an appointment
2. Schedule reminders:
```bash
curl -X POST "http://localhost:8000/api/v1/reminders/schedule-appointment-reminders/{appointment_id}" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

3. Manually trigger sending (for testing):
```bash
curl -X POST "http://localhost:8000/api/v1/reminders/send-pending" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## Reminder Schedule

When an appointment is created, reminders are automatically scheduled:

| Time Before Appointment | Channel | Purpose |
|------------------------|---------|---------|
| 24 hours | Email | Detailed reminder with appointment info |
| 2 hours | WhatsApp | Quick reminder on mobile |
| 30 minutes | SMS | Final reminder |

---

## Production Deployment

### SendGrid
- Upgrade to paid plan for more emails
- Set up domain authentication for better deliverability
- Configure webhook for tracking email events

### Twilio
- Apply for WhatsApp Business API approval
- Purchase dedicated phone number
- Set up number pool for high volume
- Configure status callbacks

### Automated Sending
Set up a cron job or scheduled task to run every 5 minutes:

```bash
# Crontab entry
*/5 * * * * curl -X POST "http://localhost:8000/api/v1/reminders/send-pending" -H "Authorization: Bearer ADMIN_TOKEN"
```

Or use a task scheduler like Celery:
```python
# celery_tasks.py
from celery import Celery

app = Celery('tasks', broker='redis://localhost:6379')

@app.task
def send_reminders():
    # Call your reminder endpoint
    pass

# Schedule to run every 5 minutes
app.conf.beat_schedule = {
    'send-reminders-every-5-minutes': {
        'task': 'tasks.send_reminders',
        'schedule': 300.0,  # 5 minutes
    },
}
```

---

## Troubleshooting

### Email Not Sending
1. Check SendGrid API key is correct
2. Verify sender email is authenticated
3. Check SendGrid dashboard for errors
4. Ensure email is not in spam folder

### WhatsApp Not Sending
1. Verify you've joined the Twilio sandbox
2. Check phone number format (+country code)
3. Ensure Twilio account has credit
4. Check Twilio logs for errors

### SMS Not Sending
1. Verify phone number is SMS-enabled
2. Check recipient number format
3. Ensure Twilio account has credit
4. Check geographic restrictions

---

## Cost Estimation

### SendGrid (Email)
- Free: 100 emails/day
- Essentials: $19.95/month - 50,000 emails
- Pro: $89.95/month - 100,000 emails

### Twilio (WhatsApp & SMS)
- WhatsApp: $0.005 per message
- SMS: $0.0075 per message (US)
- Phone number: $1/month

**Example Monthly Cost:**
- 1000 appointments/month
- 3 reminders each = 3000 notifications
- Email: Free (under 100/day)
- WhatsApp: $15
- SMS: $22.50
- **Total: ~$37.50/month**

---

## Security Best Practices

1. **Never commit .env file** to version control
2. **Rotate API keys** regularly
3. **Use environment-specific keys** (dev, staging, prod)
4. **Monitor usage** to detect anomalies
5. **Implement rate limiting** to prevent abuse

---

## Support

- SendGrid: https://support.sendgrid.com
- Twilio: https://support.twilio.com
- Documentation: See `FEATURES_IMPLEMENTED.md`

---

**Your notification system is now ready!** 🎉

Patients will receive timely reminders via email, WhatsApp, and SMS before their appointments.