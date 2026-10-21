const ones = ['', 'One', 'Two', 'Three', 'Four', 'Five', 'Six', 'Seven', 'Eight', 'Nine', 'Ten', 'Eleven', 'Twelve', 'Thirteen', 'Fourteen', 'Fifteen', 'Sixteen', 'Seventeen', 'Eighteen', 'Nineteen'];
const tens = ['', '', 'Twenty', 'Thirty', 'Forty', 'Fifty', 'Sixty', 'Seventy', 'Eighty', 'Ninety'];

function below1000(n: number): string {
	const parts: string[] = [];
	if (n >= 100) {
		parts.push(ones[Math.floor(n / 100)] + ' Hundred');
		n %= 100;
	}
	if (n >= 20) {
		parts.push(tens[Math.floor(n / 10)] + (n % 10 ? '-' + ones[n % 10] : ''));
	} else if (n > 0) {
		parts.push(ones[n]);
	}
	return parts.join(' ');
}

/** Whole number to English words. 'intl' = thousand/million/billion, 'indian' = thousand/lakh/crore. */
export function numberToWords(n: number, system: 'intl' | 'indian' = 'intl'): string {
	n = Math.floor(Math.abs(n));
	if (n === 0) return 'Zero';
	const units: [number, string][] =
		system === 'indian'
			? [[10000000, 'Crore'], [100000, 'Lakh'], [1000, 'Thousand']]
			: [[1000000000, 'Billion'], [1000000, 'Million'], [1000, 'Thousand']];
	const out: string[] = [];
	for (const [size, name] of units) {
		if (n >= size) {
			out.push(numberToWords(Math.floor(n / size), system) + ' ' + name);
			n %= size;
		}
	}
	if (n > 0) out.push(below1000(n));
	return out.join(' ');
}

const currencyNames: Record<string, [string, string]> = {
	USD: ['US Dollars', 'Cents'], EUR: ['Euro', 'Cents'], GBP: ['Pounds Sterling', 'Pence'], AED: ['UAE Dirhams', 'Fils'], SAR: ['Saudi Riyals', 'Halalas'],
	SGD: ['Singapore Dollars', 'Cents'], AUD: ['Australian Dollars', 'Cents'], CAD: ['Canadian Dollars', 'Cents'], JPY: ['Japanese Yen', ''], CNY: ['Chinese Yuan', 'Fen'],
	CHF: ['Swiss Francs', 'Rappen'], HKD: ['Hong Kong Dollars', 'Cents'], MYR: ['Malaysian Ringgit', 'Sen'], THB: ['Thai Baht', 'Satang'], ZAR: ['South African Rand', 'Cents'],
	KWD: ['Kuwaiti Dinars', 'Fils'], QAR: ['Qatari Riyals', 'Dirhams'], NZD: ['New Zealand Dollars', 'Cents'], BDT: ['Bangladeshi Taka', 'Paisa'], INR: ['Indian Rupees', 'Paise']
};

/** "US Dollars Two Hundred Fifty and Fifty Cents Only" */
export function amountInWords(amount: number | string, currency: string): string {
	const value = Math.round((typeof amount === 'string' ? parseFloat(amount) : amount) * 100) / 100;
	if (!isFinite(value)) return '';
	const [major, minor] = currencyNames[currency] || [currency, 'Cents'];
	const whole = Math.floor(value);
	const frac = Math.round((value - whole) * 100);
	let text = `${major} ${numberToWords(whole, currency === 'INR' ? 'indian' : 'intl')}`;
	if (frac > 0 && minor) text += ` and ${numberToWords(frac)} ${minor}`;
	return text + ' Only';
}
