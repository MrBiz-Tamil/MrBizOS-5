# MediConsult Hub API

A FastAPI-based backend for a medical consultation platform.

## Deployment

This project is configured for deployment on Vercel. The deployment automatically:
- Uses the Python runtime
- Installs dependencies from requirements.txt
- Serves the FastAPI app via the vercel.json configuration
- Uses SQLite with aiosqlite for serverless compatibility

## Environment Variables

Set these in your Vercel project settings:

- `DATABASE_URL`: SQLite database URL (default: sqlite+aiosqlite:///./mediconsult.db)
- `SECRET_KEY`: JWT secret key
- `ENVIRONMENT`: Set to "production" for production
- `BACKEND_CORS_ORIGINS`: CORS origins as JSON array

## Local Development

```bash
pip install -r requirements.txt
uvicorn app.main:app --reload
```
