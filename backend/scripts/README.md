# Test Users Seed Script

This script creates test users in the database for development and testing.

## 🚀 Quick Run

```bash
cd app/core-go/scripts
go run seed_users.go
```

## 👥 Test Credentials

After running the script, you can use these credentials to login:

### Admin Account
- **Email:** `admin@medihub.com`
- **Password:** `admin123`
- **Role:** Admin
- **Access:** Full system access

### Doctor Accounts

**Doctor 1:**
- **Email:** `doctor@medihub.com`
- **Password:** `doctor123`
- **Name:** Dr. John Smith
- **Specialization:** General Medicine
- **Status:** Verified, Online

**Doctor 2:**
- **Email:** `doctor2@medihub.com`
- **Password:** `doctor123`
- **Name:** Dr. Sarah Johnson
- **Specialization:** Cardiology
- **Status:** Verified, Offline

### Patient Accounts

**Patient 1:**
- **Email:** `patient@medihub.com`
- **Password:** `patient123`
- **Name:** Jane Doe

**Patient 2:**
- **Email:** `patient2@medihub.com`
- **Password:** `patient123`
- **Name:** Robert Brown

### Pharmacist Account
- **Email:** `pharmacist@medihub.com`
- **Password:** `pharmacy123`
- **Role:** Pharmacist

## 📝 Usage

### Run the Seed Script

```bash
# From project root
cd app/core-go/scripts
go run seed_users.go
```

### Expected Output

```
✅ Connected to database
🌱 Seeding test users...
✅ Created user: admin@medihub.com (admin)
✅ Created user: doctor@medihub.com (doctor)
   👨‍⚕️  Created doctor profile: General Medicine
✅ Created user: doctor2@medihub.com (doctor)
   👨‍⚕️  Created doctor profile: Cardiology
✅ Created user: patient@medihub.com (patient)
✅ Created user: patient2@medihub.com (patient)
✅ Created user: pharmacist@medihub.com (pharmacist)
✅ Seeding completed!
```

### Re-running the Script

The script is **idempotent** - it will skip users that already exist:

```
⏭️  User admin@medihub.com already exists, skipping...
⏭️  User doctor@medihub.com already exists, skipping...
```

## 🧪 Testing Login

### Using Frontend

1. Start the application:
   ```bash
   # Terminal 1
   cd app/core-go
   go run cmd/server/main.go
   
   # Terminal 2
   cd app/client
   npm run dev
   ```

2. Open browser: `http://localhost:3000/login`

3. Login with any test credentials above

### Using API Directly

```bash
# Login as admin
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@medihub.com",
    "password": "admin123"
  }'

# Login as doctor
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "doctor@medihub.com",
    "password": "doctor123"
  }'

# Login as patient
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "patient@medihub.com",
    "password": "patient123"
  }'
```

## 🔧 Customization

### Add More Users

Edit `seed_users.go` and add to the `testUsers` slice:

```go
{
    Email:    "newuser@medihub.com",
    Password: "password123",
    FullName: "New User",
    Role:     "patient",
},
```

### Change Passwords

Edit the `Password` field in the `testUsers` slice. Passwords are automatically hashed with bcrypt.

### Add Doctor Profiles

For doctor users, add a `Doctor` field:

```go
{
    Email:    "doctor3@medihub.com",
    Password: "doctor123",
    FullName: "Dr. Emily Chen",
    Role:     "doctor",
    Doctor: &DoctorProfile{
        Specialization: "Pediatrics",
        IsVerified:     true,
        IsOnline:       true,
    },
},
```

## 🗄️ Database Support

The script works with both:
- **SQLite** (default for development)
- **PostgreSQL** (production)

It reads from the `.env` file in `app/core-go/` to determine which database to use.

## ⚠️ Security Notes

**These are TEST credentials only!**

- ❌ **Never use these in production**
- ❌ **Never commit real user data**
- ✅ **Change all passwords before deploying**
- ✅ **Use strong passwords in production**
- ✅ **Enable proper authentication in production**

## 🔄 Reset Database

To start fresh:

### SQLite
```bash
# Delete the database file
rm app/core-go/medihub.db

# Run seed script again
cd app/core-go/scripts
go run seed_users.go
```

### PostgreSQL
```bash
# Drop and recreate database
psql -U postgres
DROP DATABASE medhub_prod_db;
CREATE DATABASE medhub_prod_db;
\q

# Run seed script
cd app/core-go/scripts
go run seed_users.go
```

## 📚 Related Documentation

- `QUICK_START.md` - How to run the application
- `FRONTEND_ADAPTATION.md` - Frontend setup details
- `app/core-go/README.md` - Backend documentation

---

**Last Updated:** February 13, 2026
