package hr

import (
	"fmt"
	"html"

	"github.com/gofiber/fiber/v2"
)

// Printable HR documents. These render self-contained HTML that the browser can
// print to PDF (Ctrl/Cmd-P) — avoiding a heavyweight server-side PDF dependency
// while still giving HR an issue-ready payslip / TDS certificate.

func rupees(paise int64) string {
	neg := ""
	if paise < 0 {
		neg = "-"
		paise = -paise
	}
	return fmt.Sprintf("%s₹%d.%02d", neg, paise/100, paise%100)
}

const docStyle = `<style>
body{font-family:-apple-system,Segoe UI,Roboto,sans-serif;color:#1a1a1a;max-width:720px;margin:24px auto;padding:0 24px}
h1{font-size:20px;margin:0 0 4px} .sub{color:#666;font-size:13px;margin-bottom:20px}
table{width:100%;border-collapse:collapse;margin:12px 0} td,th{padding:8px 10px;border-bottom:1px solid #eee;font-size:13px;text-align:left}
th{background:#f7f7f5;color:#555} .r{text-align:right} .tot{font-weight:700;border-top:2px solid #333}
.head{display:flex;justify-content:space-between;align-items:flex-start;border-bottom:2px solid #1a4731;padding-bottom:12px;margin-bottom:16px}
.brand{font-weight:800;color:#1a4731;font-size:18px} .muted{color:#888;font-size:12px}
@media print{.noprint{display:none}}
</style>`

// payslipDocument renders a printable salary slip.
// GET /hr/payroll/payslip-doc?user_id=&month=YYYY-MM
func payslipDocument(c *fiber.Ctx) error {
	userID := c.QueryInt("user_id")
	month := c.Query("month")
	if userID <= 0 || month == "" {
		return c.Status(400).SendString("user_id and month (YYYY-MM) are required")
	}
	ps, err := GetPayslip(uint(userID), month)
	if err != nil {
		return c.Status(404).SendString("payslip not found for this month")
	}

	doc := fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>Payslip %s</title>%s</head><body>
<div class="head"><div><div class="brand">Dr.Ayya / MediHub</div><div class="muted">Salary Slip</div></div>
<div class="muted r">Month: %s</div></div>
<h1>%s</h1><div class="sub">%s &middot; PAN: %s &middot; UAN: %s</div>
<table>
<tr><th>Earnings</th><th class="r">Amount</th><th>Deductions</th><th class="r">Amount</th></tr>
<tr><td>Basic</td><td class="r">%s</td><td>PF (employee)</td><td class="r">%s</td></tr>
<tr><td>HRA</td><td class="r">%s</td><td>ESI (employee)</td><td class="r">%s</td></tr>
<tr><td>Special Allowance</td><td class="r">%s</td><td>Professional Tax</td><td class="r">%s</td></tr>
<tr><td>&nbsp;</td><td class="r"></td><td>TDS</td><td class="r">%s</td></tr>
<tr class="tot"><td>Gross</td><td class="r">%s</td><td>Total Deductions</td><td class="r">%s</td></tr>
</table>
<table><tr class="tot"><td>Net Pay</td><td class="r">%s</td></tr>
<tr><td class="muted">Paid days</td><td class="r muted">%d</td></tr>
<tr><td class="muted">Loss of pay (days)</td><td class="r muted">%.1f</td></tr></table>
<p class="muted">This is a computer-generated payslip and does not require a signature.</p>
<button class="noprint" onclick="window.print()">Print / Save as PDF</button>
</body></html>`,
		html.EscapeString(month), docStyle, html.EscapeString(month),
		html.EscapeString(ps.EmployeeName), html.EscapeString(ps.Designation), html.EscapeString(ps.PAN), html.EscapeString(ps.UAN),
		rupees(ps.BasicSalary), rupees(ps.PFEmployee),
		rupees(ps.HRA), rupees(ps.ESIEmployee),
		rupees(ps.SpecialAllowance), rupees(ps.ProfTax),
		rupees(ps.IncomeTaxTDS),
		rupees(ps.GrossSalary), rupees(ps.TotalDeductions),
		rupees(ps.NetSalary), ps.PaidDays, ps.LopDays,
	)
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(doc)
}

// tdsCertificateDocument renders a printable Form 16A for a freelance doctor.
// GET /hr/payroll/tds-certificate-doc?doctor_id=&from=&to=
func tdsCertificateDocument(c *fiber.Ctx) error {
	doctorID := c.QueryInt("doctor_id")
	from := c.Query("from")
	to := c.Query("to")
	if doctorID <= 0 || from == "" || to == "" {
		return c.Status(400).SendString("doctor_id, from and to are required")
	}
	cert := computeTDSCertificate(uint(doctorID), from, to)
	panLine := cert.PAN
	if panLine == "" {
		panLine = "NOT ON FILE — capture before filing"
	}

	doc := fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>Form 16A</title>%s</head><body>
<div class="head"><div><div class="brand">Dr.Ayya / MediHub</div><div class="muted">Form 16A — TDS Certificate (Sec 194J)</div></div>
<div class="muted r">%s to %s</div></div>
<h1>%s</h1><div class="sub">PAN: %s</div>
<table>
<tr><th>Particulars</th><th class="r">Amount</th></tr>
<tr><td>Consultations</td><td class="r">%d</td></tr>
<tr><td>Amount paid / credited (gross)</td><td class="r">%s</td></tr>
<tr><td>Doctor gross (after commission &amp; GST)</td><td class="r">%s</td></tr>
<tr><td>TDS deducted @ %d%%</td><td class="r">%s</td></tr>
<tr class="tot"><td>Net paid</td><td class="r">%s</td></tr>
</table>
<p class="muted">Figures derived from completed consultations in the period using the platform's canonical revenue split. This certifies tax deducted at source under Section 194J.</p>
<button class="noprint" onclick="window.print()">Print / Save as PDF</button>
</body></html>`,
		docStyle, html.EscapeString(from), html.EscapeString(to),
		html.EscapeString(cert.DoctorName), html.EscapeString(panLine),
		cert.ConsultCount, rupees(cert.GrossPaise), rupees(cert.DoctorGross),
		cert.TDSRatePercent, rupees(cert.TDSPaise), rupees(cert.NetPaidPaise),
	)
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(doc)
}
