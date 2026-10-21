# MediHub Workflow Test Script — v2 (corrected routes)
# Tests all workflow scenarios against the running server on port 8081

$BaseURL = "http://localhost:8081/api"
$TotalTests = 0
$PassedTests = 0
$FailedTests = 0
$FailDetails = @()

function Test-Endpoint {
    param(
        [string]$Name, [string]$Method, [string]$URL,
        [string]$Body = "", [string]$Token = "", [int]$ExpectedStatus = 200
    )
    $script:TotalTests++
    $headers = @{ "Content-Type" = "application/json" }
    if ($Token) { $headers["Authorization"] = "Bearer $Token" }
    try {
        $params = @{ Uri = $URL; Method = $Method; Headers = $headers; ErrorAction = "Stop"; TimeoutSec = 30 }
        if ($Body -and $Method -ne "GET") { $params["Body"] = $Body }
        $response = Invoke-WebRequest @params -UseBasicParsing
        $status = $response.StatusCode
        $data = $response.Content | ConvertFrom-Json -ErrorAction SilentlyContinue
        if ($status -eq $ExpectedStatus) {
            $script:PassedTests++
            Write-Host "  [PASS] $Name (HTTP $status)" -ForegroundColor Green
            return @{ Pass = $true; Status = $status; Data = $data }
        } else {
            $script:FailedTests++
            Write-Host "  [FAIL] $Name - Expected $ExpectedStatus got $status" -ForegroundColor Red
            $script:FailDetails += "$Name (Expected $ExpectedStatus got $status)"
            return @{ Pass = $false; Status = $status; Data = $data }
        }
    } catch {
        $errStatus = 0; $errBody = ""
        if ($_.Exception.Response) {
            $errStatus = [int]$_.Exception.Response.StatusCode
            try { $sr = [System.IO.StreamReader]::new($_.Exception.Response.GetResponseStream()); $errBody = $sr.ReadToEnd(); $sr.Close() } catch {}
        }
        if ($errStatus -eq $ExpectedStatus) {
            $script:PassedTests++
            Write-Host "  [PASS] $Name (HTTP $errStatus)" -ForegroundColor Green
            $errData = $null; try { $errData = $errBody | ConvertFrom-Json } catch {}
            return @{ Pass = $true; Status = $errStatus; Data = $errData }
        }
        $script:FailedTests++
        $msg = if ($errStatus -gt 0) { "HTTP $errStatus" } else { $_.Exception.Message }
        Write-Host "  [FAIL] $Name - $msg" -ForegroundColor Red
        if ($errBody.Length -gt 0) { Write-Host "         $($errBody.Substring(0, [Math]::Min(120, $errBody.Length)))" -ForegroundColor DarkGray }
        $script:FailDetails += "$Name ($msg)"
        return @{ Pass = $false; Status = $errStatus }
    }
}

Write-Host "`n$('=' * 60)" -ForegroundColor Cyan
Write-Host " MEDIHUB COMPLETE WORKFLOW TESTS" -ForegroundColor Cyan
Write-Host "$('=' * 60)`n" -ForegroundColor Cyan

# ===== 0. HEALTH =====
Write-Host "--- 0. HEALTH CHECK ---" -ForegroundColor Yellow
Test-Endpoint "Health Check" "GET" "http://localhost:8081/health"

# ===== 1. AUTH & IDENTITY =====
Write-Host "`n--- 1. AUTH & IDENTITY ---" -ForegroundColor Yellow
$r = Test-Endpoint "Patient Login" "POST" "$BaseURL/auth/login" '{"email":"patient1@medihub.com","password":"patient123"}'
$PT = if ($r.Data.data.access_token) { $r.Data.data.access_token } else { "" }
Test-Endpoint "Patient Profile (me)" "GET" "$BaseURL/auth/me" -Token $PT

$r = Test-Endpoint "Doctor Login" "POST" "$BaseURL/auth/login" '{"email":"dr.sharma@medihub.com","password":"doctor123"}'
$DT = if ($r.Data.data.access_token) { $r.Data.data.access_token } else { "" }
Test-Endpoint "Doctor Profile (me)" "GET" "$BaseURL/auth/me" -Token $DT

$r = Test-Endpoint "Admin Login" "POST" "$BaseURL/auth/login" '{"email":"admin@medihub.com","password":"admin123"}'
$AT = if ($r.Data.data.access_token) { $r.Data.data.access_token } else { "" }

$r = Test-Endpoint "Finance Login" "POST" "$BaseURL/auth/login" '{"email":"finance@medihub.com","password":"finance123"}'
$FT = if ($r.Data.data.access_token) { $r.Data.data.access_token } else { "" }

$r = Test-Endpoint "HR Login" "POST" "$BaseURL/auth/login" '{"email":"hr@medihub.com","password":"hr123"}'
$HT = if ($r.Data.data.access_token) { $r.Data.data.access_token } else { "" }

$r = Test-Endpoint "Pharmacist Login" "POST" "$BaseURL/auth/login" '{"email":"pharmacist@medihub.com","password":"pharmacy123"}'
$PhT = if ($r.Data.data.access_token) { $r.Data.data.access_token } else { "" }

$r = Test-Endpoint "Lab Tech Login" "POST" "$BaseURL/auth/login" '{"email":"labtech@medihub.com","password":"lab123"}'
$LT = if ($r.Data.data.access_token) { $r.Data.data.access_token } else { "" }

# ===== 2. DOCTORS =====
Write-Host "`n--- 2. DOCTORS ---" -ForegroundColor Yellow
Test-Endpoint "List Doctors" "GET" "$BaseURL/doctors" -Token $PT
Test-Endpoint "Doctor Details" "GET" "$BaseURL/doctors/4" -Token $PT
Test-Endpoint "Doctor Waiting Room" "GET" "$BaseURL/doctors/waiting-room" -Token $DT
Test-Endpoint "Doctor Stats" "GET" "$BaseURL/doctors/stats/overview" -Token $DT

# ===== 3. APPOINTMENTS =====
Write-Host "`n--- 3. APPOINTMENTS ---" -ForegroundColor Yellow
Test-Endpoint "Upcoming Appointments" "GET" "$BaseURL/appointments" -Token $PT
Test-Endpoint "Past Appointments" "GET" "$BaseURL/appointments/past" -Token $PT
Test-Endpoint "Book Appointment" "POST" "$BaseURL/appointments" -Token $PT `
    -Body '{"patient_id":12,"doctor_id":4,"start_time":"2026-03-16T10:00:00Z","end_time":"2026-03-16T10:30:00Z","appointment_type":"consultation","consultation_mode":"video","notes":"Workflow test booking"}' `
    -ExpectedStatus 201

# ===== 4. EMR =====
Write-Host "`n--- 4. EMR ---" -ForegroundColor Yellow
Test-Endpoint "Prescriptions" "GET" "$BaseURL/emr/prescriptions" -Token $PT
Test-Endpoint "Diagnosis History" "GET" "$BaseURL/emr/diagnosis/history?patient_id=12" -Token $PT
Test-Endpoint "Medical Records" "GET" "$BaseURL/emr/medical-records" -Token $PT
Test-Endpoint "My Medical Records" "GET" "$BaseURL/emr/medical-records/me" -Token $PT

# ===== 5. BILLING =====
Write-Host "`n--- 5. BILLING ---" -ForegroundColor Yellow
Test-Endpoint "Billing Config" "GET" "$BaseURL/billing/config" -Token $PT

# ===== 6. LAB =====
Write-Host "`n--- 6. LAB ---" -ForegroundColor Yellow
Test-Endpoint "Lab Tests Catalog" "GET" "$BaseURL/lab/tests"
Test-Endpoint "Lab Reports" "GET" "$BaseURL/lab/reports" -Token $PT

# ===== 7. PHARMACY =====
Write-Host "`n--- 7. PHARMACY ---" -ForegroundColor Yellow
Test-Endpoint "List Pharmacies" "GET" "$BaseURL/pharmacy/" -Token $PhT
Test-Endpoint "Pharmacy Inventory" "GET" "$BaseURL/pharmacy/inventory" -Token $PhT
Test-Endpoint "Medicines Catalog" "GET" "$BaseURL/pharmacy/medicines" -Token $PhT
Test-Endpoint "Pharmacy Suppliers" "GET" "$BaseURL/pharmacy/suppliers" -Token $PhT
Test-Endpoint "Low Stock Alerts" "GET" "$BaseURL/pharmacy/inventory/low-stock" -Token $PhT
Test-Endpoint "Stock Alerts" "GET" "$BaseURL/pharmacy/alerts" -Token $PhT

# ===== 8. HOSPITAL =====
Write-Host "`n--- 8. HOSPITAL ---" -ForegroundColor Yellow
Test-Endpoint "List Hospitals" "GET" "$BaseURL/hospitals" -Token $DT
Test-Endpoint "Hospital Dashboard" "GET" "$BaseURL/hospital/dashboard" -Token $DT
Test-Endpoint "OPD Queue" "GET" "$BaseURL/hospitals/1/opd" -Token $DT
Test-Endpoint "Available Beds" "GET" "$BaseURL/hospitals/1/beds/available" -Token $DT

# ===== 9. HR =====
Write-Host "`n--- 9. HR ---" -ForegroundColor Yellow
Test-Endpoint "HR Roster" "GET" "$BaseURL/hr/roster" -Token $HT
Test-Endpoint "HR Dashboard Overview" "GET" "$BaseURL/hr/dashboard/overview" -Token $HT
Test-Endpoint "HR Staff Directory" "GET" "$BaseURL/hr/dashboard/staff" -Token $HT
Test-Endpoint "Online Doctors" "GET" "$BaseURL/hr/dashboard/doctors/online" -Token $HT

# ===== 10. COMMUNICATION =====
Write-Host "`n--- 10. COMMUNICATION ---" -ForegroundColor Yellow
Test-Endpoint "Active Consultations" "GET" "$BaseURL/communication/chat/consultations" -Token $PT
Test-Endpoint "Notifications" "GET" "$BaseURL/communication/notifications" -Token $PT

# ===== 11. STORE =====
Write-Host "`n--- 11. WELLNESS STORE ---" -ForegroundColor Yellow
Test-Endpoint "Store Products" "GET" "$BaseURL/store/products"
Test-Endpoint "Store Categories" "GET" "$BaseURL/store/categories"
Test-Endpoint "Store Orders" "GET" "$BaseURL/store/orders" -Token $PT

# ===== 12. FINANCE =====
Write-Host "`n--- 12. FINANCE ---" -ForegroundColor Yellow
Test-Endpoint "Finance Balance" "GET" "$BaseURL/finance/balance" -Token $FT
Test-Endpoint "Finance Transactions" "GET" "$BaseURL/finance/transactions" -Token $FT
Test-Endpoint "Finance Dashboard" "GET" "$BaseURL/finance/dashboard/overview" -Token $FT

# ===== 13. INSURANCE =====
Write-Host "`n--- 13. INSURANCE ---" -ForegroundColor Yellow
Test-Endpoint "Insurance Claims" "GET" "$BaseURL/insurance/claims" -Token $PT
Test-Endpoint "Insurance TPAs" "GET" "$BaseURL/insurance/tpas" -Token $PT

# ===== 14. SUPPORT =====
Write-Host "`n--- 14. SUPPORT ---" -ForegroundColor Yellow
Test-Endpoint "Support Tickets" "GET" "$BaseURL/support/tickets" -Token $PT

# ===== 15. ADMIN =====
Write-Host "`n--- 15. ADMIN ---" -ForegroundColor Yellow
Test-Endpoint "Admin Dashboard" "GET" "$BaseURL/admin/dashboard/overview" -Token $AT

# ===== SUMMARY =====
Write-Host "`n$('=' * 60)" -ForegroundColor Cyan
Write-Host " TEST RESULTS SUMMARY" -ForegroundColor Cyan
Write-Host "$('=' * 60)" -ForegroundColor Cyan
Write-Host "  Total:   $TotalTests"
$c = if ($PassedTests -eq $TotalTests) { "Green" } else { "White" }
Write-Host "  Passed:  $PassedTests" -ForegroundColor Green
Write-Host "  Failed:  $FailedTests" -ForegroundColor $(if ($FailedTests -gt 0) { "Red" } else { "Green" })
$pct = if ($TotalTests -gt 0) { [math]::Round(($PassedTests / $TotalTests) * 100, 1) } else { 0 }
Write-Host "  Rate:    $pct%"
if ($FailDetails.Count -gt 0) {
    Write-Host "`n  Failed tests:" -ForegroundColor Red
    foreach ($f in $FailDetails) { Write-Host "    - $f" -ForegroundColor DarkRed }
}
Write-Host "$('=' * 60)`n" -ForegroundColor Cyan
