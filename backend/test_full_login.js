const axios = require('axios');

async function testFullLogin() {
    console.log("Testing Login with full payload (metrics)...");
    try {
        const resp = await axios.post('http://localhost:8081/api/v1/auth/staff/login', {
            email: "doctor@medihub.com",
            password: "test123",
            behavioral_metrics: {
                typing_speed: 100,
                flight_time_mean: 100,
                mouse_velocity_mean: 100,
                timestamp: Date.now()
            }
        });
        console.log("✅ Success:", JSON.stringify(resp.data));
    } catch (err) {
        console.error("❌ Failed:", err.response?.status, JSON.stringify(err.response?.data));
    }
}

testFullLogin();
