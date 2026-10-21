const axios = require('axios');

async function testLogin(email, role) {
    console.log(`Testing Login for ${role} (${email})...`);
    try {
        const resp = await axios.post('http://localhost:8081/api/v1/auth/staff/login', {
            email: email,
            password: 'test123'
        });
        console.log(`✅ ${role} Success:`, JSON.stringify(resp.data));
    } catch (err) {
        console.error(`❌ ${role} Failed:`, err.response?.status, JSON.stringify(err.response?.data));
    }
}

async function run() {
    await testLogin('doctor@medihub.com', 'Doctor');
    await testLogin('admin@medihub.com', 'Admin');
    await testLogin('hr@medihub.com', 'HR');
    await testLogin('finance@medihub.com', 'Finance');
}

run();
