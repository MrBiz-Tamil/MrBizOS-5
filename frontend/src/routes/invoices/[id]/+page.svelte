<script lang="ts">
	import { onMount, untrack, tick } from 'svelte';
	import { page } from '$app/stores';
	import { api, setSelectedTenantId, type Invoice, type Tenant, type JournalEntry, type WhatsAppShareInfo } from '$lib/api';
	import InvoiceTemplates from '$lib/components/InvoiceTemplates.svelte';

	let invoiceId = $derived($page.params.id);
	let inv = $state<Invoice | null>(null);
	let company = $state<Tenant | null>(null);
	let journalEntry = $state<JournalEntry | null>(null);
	let loading = $state(true);
	let error = $state('');

	// Template & Print State (Template is fixed by Super Admin in Company profile)
	let selectedTemplate = $state('1');
	let selectedCopy = $state<'ORIGINAL' | 'DUPLICATE' | 'TRIPLICATE' | 'OFFICE'>('ORIGINAL');
	let selectedTaxMode = $state<'WITH_GST' | 'WITHOUT_GST'>('WITH_GST');
	let selectedPaperSize = $state<'A4' | 'A5' | 'A5_LANDSCAPE'>('A4');
	let showExtraDetailsModal = $state(false);
	let extraDetails = $state({
		poNumber: '',
		vehicleNumber: '',
		ewayBill: '',
		transporterName: '',
		lrNumber: '',
		deliveryPlace: '',
		terms: '',
		jurisdictionCity: '',
		interestRate: '',
		returnPolicy: '',
		reverseCharge: false,
		subscriptionPeriodStart: '',
		subscriptionPeriodEnd: '',
		grossWeight: '',
		netWeight: '',
		cashierName: '',
		counterNumber: '',
		projectName: '',
		signatoryTitle: '',
		signatureUrl: '',
		showBankDetails: true,
		showUPIQR: true,
		showTaxSummary: true
	});

	const templateList = [
		{ id: '1', name: '1. Modern Corporate', desc: 'Standard B2B dual-tone enterprise layout' },
		{ id: '2', name: '2. Classic Traditional', desc: 'Indian wholesale boxed grid format' },
		{ id: '3', name: '3. Minimalist Executive', desc: 'Whitespace & clean typography for consulting' },
		{ id: '4', name: '4. Retail Counter POS', desc: 'Space-saving counter receipt format' },
		{ id: '5', name: '5. Thermal 80mm Slip', desc: 'Continuous 3-inch roll receipt' },
		{ id: '6', name: '6. Creative Studio', desc: 'Vibrant agency scope of work layout' },
		{ id: '7', name: '7. Industrial & Dispatch', desc: 'Manufacturing with PO/E-Way/Vehicle' },
		{ id: '8', name: '8. Bold Tech SaaS', desc: 'Subscription receipt with cloud hero banner' },
		{ id: '9', name: '9. Wholesale Multi-Tax', desc: 'HSN/SAC summary with split CGST/SGST/IGST' },
		{ id: '10', name: '10. Simple Bill of Supply', desc: 'Non-GST / composition cash memo' },
		{ id: '11', name: '11. Industrial BlueMetals & Transport', desc: 'Heavy boxed grid with PO, Vehicle, Delivery Place, & E.& O.E' },
		{ id: '12', name: '12. Hostel / Lodging Guest Folio', desc: 'Room/bed, check-in/out, nights, GST slab summary' },
		{ id: '13', name: '13. Restaurant / Cafe Bill', desc: 'Table, waiter, KOT no, FSSAI licence, CGST/SGST split' },
		{ id: '14', name: '14. Pharmacy / Medical Store', desc: 'Drug licence, patient & doctor, batch/expiry/MRP, Schedule H notice' },
		{ id: '15', name: '15. Bakery / Sweet Shop', desc: 'FSSAI licence, veg/non-veg marks, best-before, order pickup, advance & balance' },
		{ id: '16', name: '16. Grocery / Supermarket', desc: 'MRP vs price, savings, weighed qty, GST summary incl. exempt staples' },
		{ id: '17', name: '17. Hardware / Building Materials', desc: 'Trade discount, site delivery, transport, HSN summary, credit terms' }
	];

	// Payment Modal State
	let showPaymentModal = $state(false);
	let paymentAmount = $state('');
	let paymentMethod = $state('BANK_TRANSFER');
	let paymentReference = $state('');
	let paymentNotes = $state('');
	let paymentError = $state('');
	let paymentSubmitting = $state(false);

	// Customer link
	let shareUrl = $state('');
	let shareMsg = $state('');
	async function createShare() {
		if (!inv) return;
		shareMsg = '';
		try {
			const r = await api.createShareLink(inv.id, 30);
			shareUrl = window.location.origin + r.path;
			try { await navigator.clipboard.writeText(shareUrl); shareMsg = 'Link copied. It works for 30 days.'; } catch { shareMsg = 'Copy this link. It works for 30 days.'; }
		} catch (e: any) {
			shareMsg = e.message || 'Could not create the link';
		}
	}
	async function revokeShare() {
		if (!inv) return;
		try {
			const r = await api.revokeShareLinks(inv.id);
			shareUrl = '';
			shareMsg = `${r.revoked} link(s) turned off.`;
		} catch (e: any) {
			shareMsg = e.message;
		}
	}

	// Repeat (recurring) modal
	let showRepeatModal = $state(false);
	let repeatFrequency = $state('MONTHLY');
	let repeatStart = $state('');
	let repeatError = $state('');
	let repeatBusy = $state(false);
	async function submitRepeat(e: Event) {
		e.preventDefault();
		if (!inv) return;
		repeatBusy = true;
		repeatError = '';
		try {
			await api.makeRecurring(inv.id, { frequency: repeatFrequency, start_date: repeatStart });
			window.location.href = '/invoices/recurring';
		} catch (err: any) {
			repeatError = err.message || 'Could not create the schedule';
		} finally {
			repeatBusy = false;
		}
	}

	// Cancel Modal State
	let showCancelModal = $state(false);
	let cancelReason = $state('');
	let cancelSubmitting = $state(false);

	// WhatsApp & Email States
	let whatsappLoading = $state(false);
	let whatsappShareData = $state<WhatsAppShareInfo | null>(null);
	let showWhatsAppModal = $state(false);

	let showEmailModal = $state(false);
	let emailRecipient = $state('');
	let emailSubject = $state('');
	let emailMessage = $state('');
	let emailAttachPdf = $state(true);
	let emailSubmitting = $state(false);
	let emailResultMsg = $state<{ type: 'success' | 'error'; text: string; mailto_link?: string } | null>(null);

	// Accounting Journal State
	let showJournal = $state(false);

	// PDF Download State & Handler
	let pdfDownloading = $state(false);
	async function handleDownloadPDF(size: 'a4' | 'a5') {
		if (!inv) return;
		pdfDownloading = true;
		try {
			await api.downloadInvoicePDF(inv.id, `${inv.invoice_number || 'invoice'}_${size}.pdf`, {
				copy: selectedCopy.toLowerCase(),
				tax_mode: selectedTaxMode === 'WITH_GST' ? 'with_gst' : 'without_gst',
				size: size
			});
		} catch (e: any) {
			alert('Failed to download PDF: ' + (e.message || e));
		} finally {
			pdfDownloading = false;
		}
	}


	// Print mode: used by the server (headless Chromium) to produce the PDF from this very page. The query string
	// carries the paper size, copy type and tax mode; the page flags data-print-ready once everything has rendered.
	const printMode = typeof window !== 'undefined' && new URLSearchParams(window.location.search).get('print') === '1';
	async function signalPrintReady() {
		if (!printMode) return;
		if (error || !inv) {
			document.documentElement.dataset.printError = error || 'invoice not loaded';
			return;
		}
		await tick();
		try { await document.fonts.ready; } catch {}
		await new Promise((r) => setTimeout(r, 150));
		document.documentElement.dataset.printReady = '1';
	}
	// Called by the PDF renderer with print media emulated: if the invoice is a little taller than one page, shrink it
	// to fit instead of spilling the signature block onto a second page. Long invoices still flow onto more pages.
	if (typeof window !== 'undefined') {
		(window as any).__fitToPage = () => {
			const el = document.querySelector('.invoice-print-wrapper') as HTMLElement | null;
			if (!el) return 'no sheet';
			const usableMm = selectedPaperSize === 'A5' ? 198 : selectedPaperSize === 'A5_LANDSCAPE' ? 138 : 281;
			const pageH = usableMm * 3.7795 * 0.985;
			const h = el.getBoundingClientRect().height;
			if (h > pageH && h <= pageH * 1.4) (el.style as any).zoom = String(pageH / h);
			return `content ${Math.round(h)}px of ${Math.round(pageH)}px`;
		};
	}
	if (printMode) {
		const q = new URLSearchParams(window.location.search);
		const t = q.get('tenant');
		if (t) setSelectedTenantId(t);
		const sz = (q.get('size') || 'a4').toLowerCase();
		selectedPaperSize = sz === 'a5' ? 'A5' : sz === 'a5_landscape' ? 'A5_LANDSCAPE' : 'A4';
		const cp = (q.get('copy') || 'original').toUpperCase();
		if (['ORIGINAL', 'DUPLICATE', 'TRIPLICATE', 'OFFICE'].includes(cp)) selectedCopy = cp as any;
		selectedTaxMode = q.get('tax_mode') === 'without_gst' ? 'WITHOUT_GST' : 'WITH_GST';
	}

	onMount(async () => {
		if (typeof window !== 'undefined') {
			const savedSize = localStorage.getItem('saas_default_paper_size') as any;
			if (!printMode && (savedSize === 'A4' || savedSize === 'A5' || savedSize === 'A5_LANDSCAPE')) {
				selectedPaperSize = savedSize;
			}
		}
	});

	let loadedFor = '';
	$effect(() => {
		const id = invoiceId;
		if (id && id !== loadedFor) {
			loadedFor = id;
			untrack(() => loadInvoice());
		}
	});

	function resetExtraDetails() {
		Object.assign(extraDetails, {
			poNumber: '', vehicleNumber: '', ewayBill: '', transporterName: '', lrNumber: '', deliveryPlace: '',
			terms: '', jurisdictionCity: '', interestRate: '', returnPolicy: '', reverseCharge: false,
			subscriptionPeriodStart: '', subscriptionPeriodEnd: '', grossWeight: '', netWeight: '',
			cashierName: '', counterNumber: '', projectName: ''
		});
	}

	async function loadInvoice() {
		if (!invoiceId) {
			error = 'Invalid invoice ID';
			loading = false;
			return;
		}
		loading = true;
		error = '';
		resetExtraDetails();
		try {
			const [invData, compData] = await Promise.all([
				api.getInvoice(invoiceId),
				api.getCompany()
			]);
			inv = invData;
			company = compData;
			if (compData && compData.template_id) {
				selectedTemplate = compData.template_id;
			}
			paymentAmount = invData.balance_due;

			// 1. Populate logistics directly from the DB invoice model
			if (invData.vehicle_number) extraDetails.vehicleNumber = invData.vehicle_number;
			if (invData.delivery_place) extraDetails.deliveryPlace = invData.delivery_place;
			if (invData.payment_mode) extraDetails.terms = invData.payment_mode;
			if (invData.po_number) extraDetails.poNumber = invData.po_number;
			if (invData.eway_bill) extraDetails.ewayBill = invData.eway_bill;
			if (invData.transporter_name) extraDetails.transporterName = invData.transporter_name;
			if (invData.lr_number) extraDetails.lrNumber = invData.lr_number;
			if (invData.jurisdiction_city) extraDetails.jurisdictionCity = invData.jurisdiction_city;
			if (invData.interest_rate) extraDetails.interestRate = invData.interest_rate;
			if (invData.return_policy) extraDetails.returnPolicy = invData.return_policy;
			extraDetails.reverseCharge = invData.reverse_charge || false;
			extraDetails.subscriptionPeriodStart = invData.subscription_period_start || '';
			extraDetails.subscriptionPeriodEnd = invData.subscription_period_end || '';
			extraDetails.grossWeight = invData.gross_weight || '';
			extraDetails.netWeight = invData.net_weight || '';
			extraDetails.cashierName = invData.cashier_name || '';
			extraDetails.counterNumber = invData.counter_number || '';
			extraDetails.projectName = invData.project_name || '';
			extraDetails.showBankDetails = invData.template_options?.show_bank_details ?? compData.template_options?.show_bank_details ?? true;
			extraDetails.showUPIQR = invData.template_options?.show_upi_qr ?? compData.template_options?.show_upi_qr ?? true;
			extraDetails.showTaxSummary = invData.template_options?.show_tax_summary ?? compData.template_options?.show_tax_summary ?? true;
			extraDetails.signatoryTitle = compData.signatory_title || 'Authorized Signatory';
			extraDetails.signatureUrl = compData.signature_url || '';

			// 2. Fallback to localStorage if any field was blank
			if (typeof window !== 'undefined') {
				const saved = localStorage.getItem(`saas_invoice_transport_${invoiceId}`);
				if (saved) {
					try {
						const parsed = JSON.parse(saved);
						if (!extraDetails.vehicleNumber && parsed.vehicleNumber) extraDetails.vehicleNumber = parsed.vehicleNumber;
						if (!extraDetails.deliveryPlace && parsed.deliveryPlace) extraDetails.deliveryPlace = parsed.deliveryPlace;
						if (!extraDetails.terms && (parsed.terms || parsed.paymentMode)) extraDetails.terms = parsed.terms || parsed.paymentMode;
						if (!extraDetails.poNumber && parsed.poNumber) extraDetails.poNumber = parsed.poNumber;
						if (!extraDetails.ewayBill && parsed.ewayBill) extraDetails.ewayBill = parsed.ewayBill;
						if (!extraDetails.transporterName && parsed.transporterName) extraDetails.transporterName = parsed.transporterName;
						if (!extraDetails.lrNumber && parsed.lrNumber) extraDetails.lrNumber = parsed.lrNumber;
						if (!extraDetails.jurisdictionCity && parsed.jurisdictionCity) extraDetails.jurisdictionCity = parsed.jurisdictionCity;
						if (!extraDetails.interestRate && parsed.interestRate) extraDetails.interestRate = parsed.interestRate;
						if (!extraDetails.returnPolicy && parsed.returnPolicy) extraDetails.returnPolicy = parsed.returnPolicy;
					} catch (e) {}
				}
			}

			// 3. Fallback to notes parsing if present
			if (invData.notes) {
				parseNotesForLogistics(invData.notes);
			}
			if (compData.city && !extraDetails.jurisdictionCity) {
				extraDetails.jurisdictionCity = compData.city;
			}
			if (invData.supply_state && !extraDetails.deliveryPlace) {
				extraDetails.deliveryPlace = invData.supply_state;
			}

			// Fetch double-entry journal entry if posted
			try {
				journalEntry = await api.getJournalByReference('INVOICE', invoiceId);
			} catch (jErr) {
				journalEntry = null;
			}
		} catch (err: any) {
			error = err.message || 'Failed to load invoice';
		} finally {
			loading = false;
			void signalPrintReady();
		}
	}

	function parseNotesForLogistics(notesText: string) {
		const lines = notesText.split('\n');
		for (const rawLine of lines) {
			const line = rawLine.trim();
			const lower = line.toLowerCase();
			if (lower.startsWith('vehicle:') || lower.startsWith('vehicle no:')) {
				extraDetails.vehicleNumber = line.substring(line.indexOf(':') + 1).trim();
			} else if (lower.startsWith('delivery place:') || lower.startsWith('place:')) {
				extraDetails.deliveryPlace = line.substring(line.indexOf(':') + 1).trim();
			} else if (lower.startsWith('po:') || lower.startsWith('po no:') || lower.startsWith('po number:')) {
				extraDetails.poNumber = line.substring(line.indexOf(':') + 1).trim();
			} else if (lower.startsWith('terms:') || lower.startsWith('payment mode:')) {
				extraDetails.terms = line.substring(line.indexOf(':') + 1).trim();
			} else if (lower.startsWith('jurisdiction:')) {
				extraDetails.jurisdictionCity = line.substring(line.indexOf(':') + 1).trim();
			} else if (lower.startsWith('interest:')) {
				extraDetails.interestRate = line.substring(line.indexOf(':') + 1).trim();
			} else if (lower.startsWith('policy:') || lower.startsWith('return:')) {
				extraDetails.returnPolicy = line.substring(line.indexOf(':') + 1).trim();
			} else if (lower.startsWith('eway:') || lower.startsWith('eway bill:')) {
				extraDetails.ewayBill = line.substring(line.indexOf(':') + 1).trim();
			} else if (lower.startsWith('transporter:')) {
				extraDetails.transporterName = line.substring(line.indexOf(':') + 1).trim();
			}
		}
	}

	async function saveExtraDetails() {
		if (typeof window !== 'undefined' && invoiceId) {
			localStorage.setItem(`saas_invoice_transport_${invoiceId}`, JSON.stringify(extraDetails));
		}
		if (invoiceId) {
			try {
				const updated = await api.updateInvoiceLogistics(invoiceId, {
					vehicle_number: extraDetails.vehicleNumber,
					delivery_place: extraDetails.deliveryPlace,
					payment_mode: extraDetails.terms,
					po_number: extraDetails.poNumber,
					eway_bill: extraDetails.ewayBill,
					transporter_name: extraDetails.transporterName,
					lr_number: extraDetails.lrNumber,
					jurisdiction_city: extraDetails.jurisdictionCity,
					interest_rate: extraDetails.interestRate,
					return_policy: extraDetails.returnPolicy,
					reverse_charge: extraDetails.reverseCharge,
					subscription_period_start: extraDetails.subscriptionPeriodStart,
					subscription_period_end: extraDetails.subscriptionPeriodEnd,
					gross_weight: extraDetails.grossWeight,
					net_weight: extraDetails.netWeight,
					cashier_name: extraDetails.cashierName,
					counter_number: extraDetails.counterNumber,
					project_name: extraDetails.projectName,
					template_options: {
						...(inv?.template_options || {}),
						show_bank_details: extraDetails.showBankDetails,
						show_upi_qr: extraDetails.showUPIQR,
						show_tax_summary: extraDetails.showTaxSummary
					}
				});
				if (updated) {
					inv = updated;
				}
			} catch (e: any) {
				console.warn('Failed to persist logistics to server:', e);
			}
		}
		showExtraDetailsModal = false;
	}

	async function handleWhatsAppShare() {
		if (!invoiceId) return;
		whatsappLoading = true;
		try {
			const share = await api.getWhatsAppShare(invoiceId);
			whatsappShareData = share;
			// Open WhatsApp Web directly in a new tab for Google Chrome
			if (typeof window !== 'undefined') {
				window.open(share.whatsapp_web_url, '_blank', 'noopener,noreferrer');
			}
			showWhatsAppModal = true;
		} catch (err: any) {
			alert('Failed to launch WhatsApp Web: ' + (err.message || err));
		} finally {
			whatsappLoading = false;
		}
	}

	function openEmailModal() {
		if (!inv) return;
		emailRecipient = '';
		emailSubject = `Tax Invoice ${inv.invoice_number} from ${company?.name || 'Us'}`;
		emailMessage = `Dear ${inv.customer_name || 'Valued Customer'},\n\nPlease find attached the tax invoice for your recent order.\nInvoice Number: ${inv.invoice_number}\nTotal Amount: ₹${inv.grand_total}\nBalance Due: ₹${inv.balance_due}\nDue Date: ${inv.due_date}\n\nThank you for your business!`;
		emailResultMsg = null;
		showEmailModal = true;
	}

	async function handleSendEmail(e: Event) {
		e.preventDefault();
		if (!invoiceId) return;
		emailSubmitting = true;
		emailResultMsg = null;
		try {
			const res = await api.sendInvoiceEmail(invoiceId, {
				recipient_email: emailRecipient,
				subject: emailSubject,
				custom_message: emailMessage,
				attach_pdf: emailAttachPdf
			});
			emailResultMsg = {
				type: 'success',
				text: res.message || 'Email dispatched successfully!',
				mailto_link: res.mailto_url
			};
		} catch (err: any) {
			emailResultMsg = { type: 'error', text: err.message || 'Failed to dispatch email' };
		} finally {
			emailSubmitting = false;
		}
	}

	async function handleRecordPayment(e: Event) {
		e.preventDefault();
		if (!invoiceId) return;
		paymentError = '';
		const amt = parseFloat(paymentAmount);
		if (isNaN(amt) || amt <= 0) {
			paymentError = 'Enter a valid amount';
			return;
		}
		if (inv && amt > parseFloat(inv.balance_due)) {
			paymentError = `Amount cannot exceed remaining balance of ₹${inv.balance_due}`;
			return;
		}

		paymentSubmitting = true;
		try {
			await api.createPayment({
				invoice_id: invoiceId,
				amount: paymentAmount,
				payment_method: paymentMethod,
				reference_number: paymentReference,
				notes: paymentNotes
			});
			showPaymentModal = false;
			await loadInvoice();
		} catch (err: any) {
			paymentError = err.message || 'Payment recording failed';
		} finally {
			paymentSubmitting = false;
		}
	}

	async function handleCancelInvoice(e: Event) {
		e.preventDefault();
		if (!invoiceId) return;
		cancelSubmitting = true;
		try {
			await api.cancelInvoice(invoiceId, cancelReason || 'Customer requested cancellation');
			showCancelModal = false;
			await loadInvoice();
		} catch (err: any) {
			alert('Failed to cancel invoice: ' + err.message);
		} finally {
			cancelSubmitting = false;
		}
	}

	function formatCurrency(val: string | number | undefined): string {
		if (!val) return '₹0.00';
		const num = typeof val === 'string' ? parseFloat(val) : val;
		return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(num);
	}
</script>

<svelte:head>
	<title>{inv ? `Invoice ${inv.invoice_number}` : 'Invoice Details'}</title>
	{#if selectedPaperSize === 'A5'}
		<style>
			@page {
				size: 148mm 210mm;
				margin: 6mm 8mm;
			}
			@media print {
				body, html {
					background: #ffffff !important;
					color: #000000 !important;
					margin: 0 !important;
					padding: 0 !important;
					-webkit-print-color-adjust: exact !important;
					print-color-adjust: exact !important;
				}
				.no-print, .template-controls, .sidebar, .topbar, header, nav, button {
					display: none !important;
				}
				.invoice-print-wrapper, .layout-container, .main-content, .page-body {
					padding: 0 !important;
					margin: 0 !important;
					width: 100% !important;
					max-width: 100% !important;
					display: block !important;
					overflow: visible !important;
				}
				.invoice-sheet-container {
					box-shadow: none !important;
					border: none !important;
					margin: 0 auto !important;
					width: 100% !important;
					max-width: 100% !important;
					page-break-inside: avoid !important;
					break-inside: avoid !important;
				}
			}
		</style>
	{:else if selectedPaperSize === 'A5_LANDSCAPE'}
		<style>
			@page {
				size: 210mm 148mm;
				margin: 5mm 6mm;
			}
			@media print {
				body, html {
					background: #ffffff !important;
					color: #000000 !important;
					margin: 0 !important;
					padding: 0 !important;
					-webkit-print-color-adjust: exact !important;
					print-color-adjust: exact !important;
				}
				.no-print, .template-controls, .sidebar, .topbar, header, nav, button {
					display: none !important;
				}
				.invoice-print-wrapper, .layout-container, .main-content, .page-body {
					padding: 0 !important;
					margin: 0 !important;
					width: 100% !important;
					max-width: 100% !important;
					display: block !important;
					overflow: visible !important;
				}
				.invoice-sheet-container {
					box-shadow: none !important;
					border: none !important;
					margin: 0 auto !important;
					width: 100% !important;
					max-width: 100% !important;
					page-break-inside: avoid !important;
					break-inside: avoid !important;
				}
			}
		</style>
	{:else}
		<style>
			@page {
				size: A4 portrait;
				margin: 8mm 10mm;
			}
			@media print {
				body, html {
					background: #ffffff !important;
					color: #000000 !important;
					margin: 0 !important;
					padding: 0 !important;
					-webkit-print-color-adjust: exact !important;
					print-color-adjust: exact !important;
				}
				.no-print, .template-controls, .sidebar, .topbar, header, nav, button {
					display: none !important;
				}
				.invoice-print-wrapper, .layout-container, .main-content, .page-body {
					padding: 0 !important;
					margin: 0 !important;
					width: 100% !important;
					max-width: 100% !important;
					display: block !important;
					overflow: visible !important;
				}
				.invoice-sheet-container {
					box-shadow: none !important;
					border: none !important;
					margin: 0 auto !important;
					width: 100% !important;
					max-width: 100% !important;
					page-break-inside: avoid !important;
					break-inside: avoid !important;
				}
			}
		</style>
	{/if}
</svelte:head>

<div class="invoice-print-wrapper" style="display: flex; flex-direction: column; gap: 1.5rem;">
	{#if shareMsg || shareUrl}
		<div class="card no-print" style="padding: 0.75rem 1rem; font-size: 0.8125rem;">
			{shareMsg}
			{#if shareUrl}
				<input class="form-input" readonly value={shareUrl} style="margin-top: 0.5rem;" onfocus={(e) => (e.target as HTMLInputElement).select()} />
				<button class="btn btn-secondary" style="margin-top: 0.5rem;" onclick={revokeShare}>Turn off all links for this invoice</button>
			{/if}
		</div>
	{/if}
	<!-- Top Bar -->
	<div class="no-print" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
		<div>
			<a href="/invoices" style="font-size: 0.8125rem; color: #818cf8; text-decoration: none; margin-bottom: 0.25rem; display: inline-block;">
				← Back to Invoices
			</a>
			<div style="display: flex; align-items: center; gap: 0.75rem;">
				<h1 style="font-size: 1.75rem; font-family: var(--font-mono);">{inv?.invoice_number || 'Invoice'}</h1>
				{#if inv}
					<span class={`badge badge-${inv.status.toLowerCase().replace('_', '-')}`}>
						<span class="badge-dot"></span>
						{inv.status}
					</span>
				{/if}
			</div>
		</div>

		{#if inv}
			<div class="template-controls no-print" style="display: flex; gap: 0.6rem; align-items: center; flex-wrap: wrap;">
				<!-- WhatsApp Web Button (Chrome Web Browser Direct) -->
				<button
					type="button"
					onclick={handleWhatsAppShare}
					class="btn"
					style="background: rgba(37, 211, 102, 0.15); color: #25d366; border: 1px solid rgba(37, 211, 102, 0.4); display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.45rem 0.85rem; font-size: 0.8125rem; font-weight: 600;"
					disabled={whatsappLoading}
					title="Share bill via WhatsApp Web directly in Google Chrome"
				>
					<span>💬</span>
					<span>{whatsappLoading ? 'Opening...' : 'Share WhatsApp'}</span>
				</button>

				<!-- Email Bill Button -->
				<button
					type="button"
					onclick={openEmailModal}
					class="btn"
					style="background: rgba(56, 189, 248, 0.15); color: #38bdf8; border: 1px solid rgba(56, 189, 248, 0.4); display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.45rem 0.85rem; font-size: 0.8125rem; font-weight: 600;"
					title="Email PDF bill & details to customer"
				>
					<span>✉️</span>
					<span>Email Bill</span>
				</button>

				<!-- Audited Accounting Voucher Toggle -->
				{#if journalEntry}
					<button
						type="button"
						onclick={() => (showJournal = !showJournal)}
						class="btn btn-secondary"
						style="display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.45rem 0.85rem; font-size: 0.8125rem; font-weight: 600; border-color: rgba(129, 140, 248, 0.4); color: #a5b4fc;"
						title="Inspect double-entry journal posting for this invoice"
					>
						<span>📖</span>
						<span>{showJournal ? 'Hide Accounting' : 'Accounting Voucher'}</span>
					</button>
				{/if}

				<!-- Print Button -->
				<button onclick={() => window.print()} class="btn btn-primary" title={`Print invoice on ${selectedPaperSize.replace('_', ' ')} paper`}>
					🖨️ Print ({selectedPaperSize === 'A5_LANDSCAPE' ? 'A5 Land' : selectedPaperSize})
				</button>

				<!-- Dedicated A4 & A5 PDF Download Buttons -->
				<div style="display: inline-flex; border: 1px solid var(--border-subtle); border-radius: var(--radius-md); overflow: hidden; box-shadow: var(--shadow-sm);">
					<button
						type="button"
						onclick={() => handleDownloadPDF('a4')}
						disabled={pdfDownloading}
						class="btn btn-secondary btn-sm"
						style="border-radius: 0; border: none; padding: 0.45rem 0.75rem; font-size: 0.8125rem; font-weight: {selectedPaperSize === 'A4' ? '700' : '500'}; background: {selectedPaperSize === 'A4' ? 'rgba(99, 102, 241, 0.25)' : 'transparent'}; color: {selectedPaperSize === 'A4' ? '#a5b4fc' : 'var(--text-secondary)'};"
						title="Download standard A4 PDF document via authenticated stream"
					>
						📄 {pdfDownloading ? 'Generating...' : 'PDF (A4)'}
					</button>
					<button
						type="button"
						onclick={() => handleDownloadPDF('a5')}
						disabled={pdfDownloading}
						class="btn btn-secondary btn-sm"
						style="border-radius: 0; border: none; border-left: 1px solid var(--border-subtle); padding: 0.45rem 0.75rem; font-size: 0.8125rem; font-weight: {selectedPaperSize !== 'A4' ? '700' : '500'}; background: {selectedPaperSize !== 'A4' ? 'rgba(16, 185, 129, 0.25)' : 'transparent'}; color: {selectedPaperSize !== 'A4' ? '#34d399' : 'var(--text-secondary)'};"
						title="Download compact A5 (Half-Sheet) PDF bill via authenticated stream"
					>
						📑 PDF (A5)
					</button>
				</div>

				{#if inv.status !== 'PAID' && inv.status !== 'CANCELLED'}
					<button onclick={() => (showPaymentModal = true)} class="btn btn-secondary" style="border-color: rgba(16, 185, 129, 0.4); color: #34d399;">
						💳 Record Payment
					</button>
				{/if}

				<button onclick={createShare} class="btn btn-secondary" title="Private link the customer can open without signing in">
					🔗 Customer link
				</button>

				<button onclick={() => { repeatStart = new Date(Date.now() + 30 * 86400000).toISOString().slice(0, 10); showRepeatModal = true; }} class="btn btn-secondary">
					🔁 Repeat this invoice
				</button>

				{#if inv.status === 'ISSUED' && parseFloat(inv.amount_paid) === 0}
					<button onclick={() => (showCancelModal = true)} class="btn btn-danger btn-sm">
						Cancel Invoice
					</button>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Interactive Copy & Paper Selector Bar (Template fixed by Super Admin) -->
	{#if inv}
		<div class="card template-controls no-print" style="padding: 1rem 1.25rem; background: rgba(17, 24, 39, 0.85); border-color: rgba(99, 102, 241, 0.3);">
			<div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
				<div style="display: flex; align-items: center; gap: 1rem; flex-wrap: wrap;">
					<!-- Paper Size Selector (A4 Standard vs A5 Portrait vs A5 Landscape) -->
					<div style="display: flex; align-items: center; gap: 0.5rem;">
						<span style="font-size: 0.75rem; font-weight: 700; color: #a5b4fc; text-transform: uppercase;">Paper Size:</span>
						<div style="display: inline-flex; border: 1px solid var(--border-subtle); border-radius: var(--radius-md); overflow: hidden;">
							<button
								type="button"
								onclick={() => { selectedPaperSize = 'A4'; if (typeof window !== 'undefined') localStorage.setItem('saas_default_paper_size', 'A4'); }}
								class="btn btn-sm"
								style="border-radius: 0; padding: 0.3rem 0.65rem; font-size: 0.75rem; background: {selectedPaperSize === 'A4' ? 'rgba(99, 102, 241, 0.3)' : 'transparent'}; color: {selectedPaperSize === 'A4' ? '#a5b4fc' : 'var(--text-secondary)'}; font-weight: {selectedPaperSize === 'A4' ? '700' : '500'};"
								title="Standard A4 sheet (210 x 297 mm)"
							>
								📄 A4 Standard
							</button>
							<button
								type="button"
								onclick={() => { selectedPaperSize = 'A5'; if (typeof window !== 'undefined') localStorage.setItem('saas_default_paper_size', 'A5'); }}
								class="btn btn-sm"
								style="border-radius: 0; padding: 0.3rem 0.65rem; font-size: 0.75rem; background: {selectedPaperSize === 'A5' ? 'rgba(16, 185, 129, 0.25)' : 'transparent'}; color: {selectedPaperSize === 'A5' ? '#34d399' : 'var(--text-secondary)'}; font-weight: {selectedPaperSize === 'A5' ? '700' : '500'};"
								title="Half-sheet A5 Portrait (148 x 210 mm) - 50% paper saving"
							>
								📑 A5 Portrait
							</button>
							<button
								type="button"
								onclick={() => { selectedPaperSize = 'A5_LANDSCAPE'; if (typeof window !== 'undefined') localStorage.setItem('saas_default_paper_size', 'A5_LANDSCAPE'); }}
								class="btn btn-sm"
								style="border-radius: 0; padding: 0.3rem 0.65rem; font-size: 0.75rem; background: {selectedPaperSize === 'A5_LANDSCAPE' ? 'rgba(56, 189, 248, 0.25)' : 'transparent'}; color: {selectedPaperSize === 'A5_LANDSCAPE' ? '#38bdf8' : 'var(--text-secondary)'}; font-weight: {selectedPaperSize === 'A5_LANDSCAPE' ? '700' : '500'};"
								title="Half-sheet A5 Landscape (210 x 148 mm) - Compact wide format"
							>
								📜 A5 Landscape
							</button>
						</div>
					</div>

					<!-- Template Selection -->
					<div style="display: flex; align-items: center; gap: 0.5rem;">
						<span style="font-size: 0.75rem; font-weight: 700; color: #a5b4fc; text-transform: uppercase;">Template:</span>
						<select
							id="select-invoice-template"
							class="form-select"
							style="width: auto; padding: 0.35rem 0.75rem; font-size: 0.8125rem; background: #1f2937; color: #e0e7ff; font-weight: 600;"
							bind:value={selectedTemplate}
						>
							{#each templateList as tpl}
								<option value={tpl.id}>{tpl.name}</option>
							{/each}
						</select>
					</div>

					<!-- Copy Type Selection -->
					<div style="display: flex; align-items: center; gap: 0.5rem;">
						<span style="font-size: 0.75rem; font-weight: 700; color: #94a3b8; text-transform: uppercase;">Copy:</span>
						<select
							class="form-select"
							style="width: auto; padding: 0.35rem 0.75rem; font-size: 0.8125rem; background: #1f2937;"
							bind:value={selectedCopy}
						>
							<option value="ORIGINAL">Original (Recipient)</option>
							<option value="DUPLICATE">Duplicate (Transporter)</option>
							<option value="TRIPLICATE">Triplicate (Supplier)</option>
							<option value="OFFICE">Office Copy</option>
						</select>
					</div>

					<!-- GST vs Non-GST Mode -->
					<div style="display: flex; align-items: center; gap: 0.5rem;">
						<span style="font-size: 0.75rem; font-weight: 700; color: #94a3b8; text-transform: uppercase;">Tax Mode:</span>
						<div style="display: inline-flex; border: 1px solid var(--border-subtle); border-radius: var(--radius-md); overflow: hidden;">
							<button
								type="button"
								onclick={() => (selectedTaxMode = 'WITH_GST')}
								class="btn btn-sm"
								style="border-radius: 0; padding: 0.3rem 0.65rem; font-size: 0.75rem; background: {selectedTaxMode === 'WITH_GST' ? 'rgba(99, 102, 241, 0.3)' : 'transparent'}; color: {selectedTaxMode === 'WITH_GST' ? '#a5b4fc' : 'var(--text-secondary)'}; font-weight: {selectedTaxMode === 'WITH_GST' ? '700' : '500'};"
							>
								With GST (Tax Inv)
							</button>
							<button
								type="button"
								onclick={() => (selectedTaxMode = 'WITHOUT_GST')}
								class="btn btn-sm"
								style="border-radius: 0; padding: 0.3rem 0.65rem; font-size: 0.75rem; background: {selectedTaxMode === 'WITHOUT_GST' ? 'rgba(245, 158, 11, 0.25)' : 'transparent'}; color: {selectedTaxMode === 'WITHOUT_GST' ? '#fbbf24' : 'var(--text-secondary)'}; font-weight: {selectedTaxMode === 'WITHOUT_GST' ? '700' : '500'};"
							>
								Without GST (Bill of Supply)
							</button>
						</div>
					</div>
				</div>

				<!-- Extra Template & Detail Options Button -->
				<div>
					<button
						id="btn-open-detail-options"
						type="button"
						onclick={() => (showExtraDetailsModal = true)}
						class="btn btn-secondary btn-sm"
						style="font-size: 0.75rem; padding: 0.35rem 0.75rem; display: inline-flex; align-items: center; gap: 0.4rem;"
					>
						<span>⚙️</span>
						<span>Template & Detail Options</span>
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Audited Double-Entry Accounting Journal Voucher Card -->
	{#if journalEntry && showJournal}
		<div class="card animate-fade-in no-print" style="background: rgba(15, 23, 42, 0.85); border: 1px solid rgba(129, 140, 248, 0.4); padding: 1.25rem;">
			<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.75rem;">
				<div>
					<div style="display: flex; align-items: center; gap: 0.5rem;">
						<span style="font-size: 1.125rem;">📖</span>
						<h3 style="font-size: 1.125rem; font-weight: 700; color: #a5b4fc; margin: 0;">
							Audited Double-Entry Journal Voucher: <span class="font-mono">{journalEntry.entry_number}</span>
						</h3>
					</div>
					<div style="font-size: 0.8125rem; color: var(--text-secondary); margin-top: 0.25rem;">
						Posted on {new Date(journalEntry.entry_date).toLocaleDateString('en-IN')} | Ref: {journalEntry.reference_type} #{inv?.invoice_number}
					</div>
				</div>
				<div style="display: flex; align-items: center; gap: 0.75rem;">
					<span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399; font-weight: 600;">
						✓ Balanced (Debit = Credit: {formatCurrency(journalEntry.total_debit)})
					</span>
					<a href="/accounting" class="btn btn-secondary btn-sm" style="font-size: 0.75rem;">
						Open General Ledger →
					</a>
				</div>
			</div>

			<div class="table-container" style="overflow-x: auto; background: rgba(0, 0, 0, 0.2); border-radius: var(--radius-md); border: 1px solid var(--border);">
				<table class="data-table" style="font-size: 0.8125rem;">
					<thead>
						<tr>
							<th style="width: 50px;">#</th>
							<th>Account Code & Title</th>
							<th>Type</th>
							<th>Sub-Ledger</th>
							<th style="text-align: right;">Debit (₹)</th>
							<th style="text-align: right;">Credit (₹)</th>
							<th>Line Narration</th>
						</tr>
					</thead>
					<tbody>
						{#each journalEntry.lines || [] as line, idx}
							<tr>
								<td class="font-mono" style="color: var(--text-muted);">{idx + 1}</td>
								<td>
									<span class="font-mono" style="font-weight: 700; color: #38bdf8;">{line.account_code || 'ACC'}</span>
									<span style="font-weight: 600; margin-left: 0.5rem;">{line.account_name || 'Account'}</span>
								</td>
								<td style="color: var(--text-secondary);">General</td>
								<td style="color: #fbbf24;">{line.customer_name || '-'}</td>
								<td class="font-mono" style="text-align: right; font-weight: 600; color: {parseFloat(line.debit_amount) > 0 ? '#34d399' : 'var(--text-muted)'};">
									{parseFloat(line.debit_amount) > 0 ? formatCurrency(line.debit_amount) : '—'}
								</td>
								<td class="font-mono" style="text-align: right; font-weight: 600; color: {parseFloat(line.credit_amount) > 0 ? '#818cf8' : 'var(--text-muted)'};">
									{parseFloat(line.credit_amount) > 0 ? formatCurrency(line.credit_amount) : '—'}
								</td>
								<td style="color: var(--text-secondary); font-size: 0.75rem;">{line.line_narration || '-'}</td>
							</tr>
						{/each}
					</tbody>
					<tfoot>
						<tr style="border-top: 2px solid var(--border); font-weight: 700; background: rgba(0,0,0,0.3);">
							<td colspan="4" style="text-align: right;">Total Equilibrium:</td>
							<td class="font-mono" style="text-align: right; color: #34d399;">{formatCurrency(journalEntry.total_debit)}</td>
							<td class="font-mono" style="text-align: right; color: #818cf8;">{formatCurrency(journalEntry.total_credit)}</td>
							<td>
								<span style="color: #34d399; font-size: 0.75rem;">✓ Double-Entry Verified</span>
							</td>
						</tr>
					</tfoot>
				</table>
			</div>
		</div>
	{/if}

	{#if loading}
		<div style="padding: 3rem; text-align: center; color: var(--text-muted);">
			Loading invoice details...
		</div>
	{:else if error}
		<div class="card" style="border-color: rgba(244, 63, 94, 0.4); color: #fb7185; padding: 2rem; text-align: center; display: flex; flex-direction: column; align-items: center; gap: 1rem;">
			<div>⚠️ {error}</div>
			<a href="/invoices" class="btn btn-secondary btn-sm">← Back to Invoices List</a>
		</div>
	{:else if inv}
		<!-- Dynamically Rendered Billing Template with Paper Size Selection -->
		<InvoiceTemplates
			invoice={inv}
			company={company}
			templateId={selectedTemplate}
			copyType={selectedCopy}
			taxMode={selectedTaxMode}
			paperSize={selectedPaperSize}
			{extraDetails}
		/>
	{/if}
</div>

<!-- WhatsApp Web Share Preview Modal -->
{#if showWhatsAppModal && whatsappShareData}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 580px;">
			<div class="modal-header">
				<div style="display: flex; align-items: center; gap: 0.5rem;">
					<span style="font-size: 1.25rem;">💬</span>
					<h3 style="font-size: 1.125rem; margin: 0; color: #25d366;">WhatsApp Web Share (Google Chrome)</h3>
				</div>
				<button onclick={() => (showWhatsAppModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>
			<div class="modal-body" style="display: flex; flex-direction: column; gap: 1rem;">
				<div style="background: rgba(37, 211, 102, 0.1); border: 1px solid rgba(37, 211, 102, 0.3); border-radius: var(--radius-md); padding: 0.75rem 1rem; font-size: 0.8125rem; color: #34d399;">
					✓ WhatsApp Web has been initiated in your browser with pre-filled invoice message and customer recipient.
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label for="wa-phone" class="form-label" style="font-size: 0.75rem;">Recipient Phone (Formatted for WhatsApp)</label>
					<input id="wa-phone" type="text" class="form-input font-mono" readonly value={`+${whatsappShareData.formatted_phone}`} />
				</div>

				<div class="form-group" style="margin-bottom: 0;">
					<label for="wa-msg" class="form-label" style="font-size: 0.75rem;">Composed Message Preview</label>
					<textarea id="wa-msg" class="form-input font-mono" rows="6" readonly style="font-size: 0.75rem; white-space: pre-wrap;" value={whatsappShareData.message_text}></textarea>
				</div>
			</div>
			<div class="modal-footer" style="display: flex; justify-content: space-between; align-items: center;">
				<button
					type="button"
					onclick={() => {
						navigator.clipboard?.writeText(whatsappShareData?.message_text || '');
						alert('WhatsApp message copied to clipboard!');
					}}
					class="btn btn-secondary btn-sm"
				>
					📋 Copy Message Text
				</button>
				<div style="display: flex; gap: 0.5rem;">
					<button type="button" onclick={() => (showWhatsAppModal = false)} class="btn btn-secondary">
						Close
					</button>
					<a
						href={whatsappShareData.whatsapp_web_url}
						target="_blank"
						rel="noopener noreferrer"
						class="btn"
						style="background: #25d366; color: #000; font-weight: 700;"
					>
						Open WhatsApp Web Again ↗
					</a>
				</div>
			</div>
		</div>
	</div>
{/if}

<!-- Send Email Modal -->
{#if showEmailModal && inv}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 580px;">
			<div class="modal-header">
				<div style="display: flex; align-items: center; gap: 0.5rem;">
					<span style="font-size: 1.25rem;">✉️</span>
					<h3 style="font-size: 1.125rem; margin: 0; color: #38bdf8;">Email Invoice & PDF Bill</h3>
				</div>
				<button onclick={() => (showEmailModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>

			<form onsubmit={handleSendEmail}>
				<div class="modal-body" style="display: flex; flex-direction: column; gap: 1rem;">
					{#if emailResultMsg}
						<div
							style="border-radius: var(--radius-md); padding: 0.75rem; font-size: 0.8125rem; {emailResultMsg.type === 'success'
								? 'background: rgba(16, 185, 129, 0.15); border: 1px solid rgba(16, 185, 129, 0.3); color: #34d399;'
								: 'background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); color: #fb7185;'}"
						>
							<div>{emailResultMsg.type === 'success' ? '✓' : '⚠️'} {emailResultMsg.text}</div>
							{#if emailResultMsg.mailto_link}
								<div style="margin-top: 0.5rem;">
									<a
										href={emailResultMsg.mailto_link}
										class="btn btn-secondary btn-sm"
										style="font-size: 0.75rem; background: rgba(56, 189, 248, 0.2); color: #38bdf8;"
									>
										Launch Desktop Email App (Outlook / Mail) ↗
									</a>
								</div>
							{/if}
						</div>
					{/if}

					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="email-to">Recipient Email Address *</label>
						<input id="email-to" type="email" class="form-input" bind:value={emailRecipient} placeholder="customer@example.com" required />
					</div>

					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="email-sub">Subject Line *</label>
						<input id="email-sub" type="text" class="form-input" bind:value={emailSubject} placeholder="Tax Invoice ..." required />
					</div>

					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="email-body">Email Body Message</label>
						<textarea id="email-body" class="form-input font-mono" rows="4" bind:value={emailMessage} style="font-size: 0.8125rem;"></textarea>
					</div>

					<div style="display: flex; align-items: center; gap: 0.5rem; background: rgba(255,255,255,0.03); padding: 0.5rem 0.75rem; border-radius: var(--radius-sm);">
						<input type="checkbox" id="attach-pdf" bind:checked={emailAttachPdf} />
						<label for="attach-pdf" style="font-size: 0.8125rem; cursor: pointer;">
							Attach PDF Bill ({selectedPaperSize === 'A4' ? 'A4 Format' : 'A5 Format'}) automatically
						</label>
					</div>
				</div>

				<div class="modal-footer">
					<button type="button" onclick={() => (showEmailModal = false)} class="btn btn-secondary">
						Close
					</button>
					<button type="submit" class="btn btn-primary" disabled={emailSubmitting}>
						{emailSubmitting ? 'Dispatching...' : 'Send Invoice Email'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Record Payment Modal -->
{#if showPaymentModal && inv}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in">
			<div class="modal-header">
				<h3 style="font-size: 1.125rem;">Record Invoice Payment</h3>
				<button onclick={() => (showPaymentModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>
			<form onsubmit={handleRecordPayment}>
				<div class="modal-body">
					{#if paymentError}
						<div style="background: rgba(244, 63, 94, 0.15); border: 1px solid rgba(244, 63, 94, 0.3); border-radius: var(--radius-md); padding: 0.75rem; color: #fb7185; font-size: 0.8125rem; margin-bottom: 1rem;">
							⚠️ {paymentError}
						</div>
					{/if}

					<div style="background: rgba(255,255,255,0.04); padding: 0.75rem 1rem; border-radius: var(--radius-md); margin-bottom: 1.25rem; display: flex; justify-content: space-between;">
						<span style="font-size: 0.8125rem; color: var(--text-secondary);">Remaining Balance Due:</span>
						<span class="font-mono" style="font-weight: 700; color: var(--amber);">{formatCurrency(inv.balance_due)}</span>
					</div>

					<div class="form-group">
						<label class="form-label" for="pay-amt">Payment Amount (₹) *</label>
						<input
							id="pay-amt"
							type="number"
							step="0.01"
							max={inv.balance_due}
							class="form-input font-mono"
							bind:value={paymentAmount}
							required
						/>
					</div>

					<div class="form-group">
						<label class="form-label" for="pay-method">Payment Method</label>
						<select id="pay-method" class="form-select" bind:value={paymentMethod}>
							<option value="BANK_TRANSFER">Bank Transfer / NEFT / RTGS</option>
							<option value="UPI">UPI (Google Pay / PhonePe / Paytm)</option>
							<option value="CASH">Cash</option>
							<option value="CARD">Debit / Credit Card</option>
							<option value="CHEQUE">Cheque</option>
							<option value="OTHER">Other</option>
						</select>
					</div>

					<div class="form-group">
						<label class="form-label" for="pay-ref">Transaction Reference / UTR Number</label>
						<input id="pay-ref" type="text" class="form-input" bind:value={paymentReference} placeholder="e.g. UTR12345678" />
					</div>

					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="pay-notes">Notes</label>
						<input id="pay-notes" type="text" class="form-input" bind:value={paymentNotes} placeholder="Optional receipt note..." />
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" onclick={() => (showPaymentModal = false)} class="btn btn-secondary">Cancel</button>
					<button type="submit" class="btn btn-primary" disabled={paymentSubmitting}>
						{paymentSubmitting ? 'Recording...' : 'Confirm Payment'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Repeat Invoice Modal -->
{#if showRepeatModal}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in">
			<div class="modal-header">
				<h3 style="font-size: 1.125rem;">Repeat this invoice automatically</h3>
				<button onclick={() => (showRepeatModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>
			<form onsubmit={submitRepeat}>
				<p style="color: var(--text-secondary); font-size: 0.875rem;">A new invoice with the same customer and items will be issued on each date.</p>
				{#if repeatError}<p style="color: #fb7185; font-size: 0.8125rem;">{repeatError}</p>{/if}
				<div class="form-group"><label class="form-label" for="rf">Repeat</label>
					<select id="rf" class="form-input" bind:value={repeatFrequency}>
						<option value="WEEKLY">Every week</option><option value="MONTHLY">Every month</option>
						<option value="QUARTERLY">Every quarter</option><option value="YEARLY">Every year</option>
					</select></div>
				<div class="form-group"><label class="form-label" for="rs">First invoice on</label>
					<input id="rs" type="date" class="form-input" bind:value={repeatStart} required /></div>
				<button class="btn btn-primary" disabled={repeatBusy}>{repeatBusy ? 'Saving...' : 'Start schedule'}</button>
			</form>
		</div>
	</div>
{/if}

<!-- Cancel Invoice Modal -->
{#if showCancelModal}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in">
			<div class="modal-header">
				<h3 style="font-size: 1.125rem; color: var(--rose);">Cancel Issued Invoice</h3>
				<button onclick={() => (showCancelModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>
			<form onsubmit={handleCancelInvoice}>
				<div class="modal-body">
					<p style="font-size: 0.875rem; color: var(--text-secondary); margin-bottom: 1rem;">
						Under strict accounting rules, issued financial invoices cannot be deleted. Cancelling this invoice will preserve the audit record, generate a balanced reversal journal entry in the General Ledger, restore inventory stock, and mark the invoice as CANCELLED.
					</p>
					<div class="form-group" style="margin-bottom: 0;">
						<label class="form-label" for="cancel-reason">Cancellation Reason *</label>
						<input id="cancel-reason" type="text" class="form-input" bind:value={cancelReason} placeholder="e.g. Customer order cancelled prior to dispatch" required />
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" onclick={() => (showCancelModal = false)} class="btn btn-secondary">Close</button>
					<button type="submit" class="btn btn-danger" disabled={cancelSubmitting}>
						{cancelSubmitting ? 'Cancelling...' : 'Confirm Cancellation'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<!-- Template & Detail Options Modal -->
{#if showExtraDetailsModal}
	<div class="modal-backdrop">
		<div class="modal-card animate-fade-in" style="max-width: 680px; max-height: 90vh; display: flex; flex-direction: column;">
			<div class="modal-header">
				<div>
					<h3 style="font-size: 1.125rem; margin-bottom: 2px;">Invoice Template & Detail Options</h3>
					<div style="font-size: 0.75rem; color: var(--text-muted);">
						Customize logistics, weights, service periods, project scope, payment terms, and visual toggles for this invoice.
					</div>
				</div>
				<button onclick={() => (showExtraDetailsModal = false)} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 1.25rem;">✕</button>
			</div>
			<div class="modal-body" style="display: flex; flex-direction: column; gap: 1.25rem; overflow-y: auto; padding: 1.25rem;">
				<!-- Logistics & Dispatch (Templates 7, 11, B2B) -->
				<div>
					<div style="font-size: 0.8125rem; font-weight: 700; color: #a5b4fc; text-transform: uppercase; margin-bottom: 0.5rem; display: flex; align-items: center; gap: 0.35rem;">
						<span>🚚</span>
						<span>Logistics, Transport & Weighment (Templates 7, 11 & B2B)</span>
					</div>
					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-veh">Vehicle / Truck Number</label>
							<input id="dtl-veh" type="text" class="form-input font-mono" bind:value={extraDetails.vehicleNumber} placeholder="TN74BA6009 or MH-12-AB-1234" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-place">Delivery Place / Site</label>
							<input id="dtl-place" type="text" class="form-input" bind:value={extraDetails.deliveryPlace} placeholder="POTHAYADI or Customer Site" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-po">Customer PO / Work Order #</label>
							<input id="dtl-po" type="text" class="form-input" bind:value={extraDetails.poNumber} placeholder="PO-12345" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-eway">E-Way Bill Number</label>
							<input id="dtl-eway" type="text" class="form-input font-mono" bind:value={extraDetails.ewayBill} placeholder="321098475621" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-trans">Transporter Name</label>
							<input id="dtl-trans" type="text" class="form-input" bind:value={extraDetails.transporterName} placeholder="BlueMetals Logistics" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-lr">LR / Consignment Note #</label>
							<input id="dtl-lr" type="text" class="form-input font-mono" bind:value={extraDetails.lrNumber} placeholder="LR-88741" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-gross-wt">Gross Weight (Dispatch)</label>
							<input id="dtl-gross-wt" type="text" class="form-input" bind:value={extraDetails.grossWeight} placeholder="e.g. 24.50 MT" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-net-wt">Net Weight</label>
							<input id="dtl-net-wt" type="text" class="form-input" bind:value={extraDetails.netWeight} placeholder="e.g. 23.80 MT" />
						</div>
					</div>
				</div>

				<hr style="border: 0; border-top: 1px solid var(--border-subtle); margin: 0;" />

				<!-- SaaS, Project & Retail Details -->
				<div>
					<div style="font-size: 0.8125rem; font-weight: 700; color: #a5b4fc; text-transform: uppercase; margin-bottom: 0.5rem; display: flex; align-items: center; gap: 0.35rem;">
						<span>💼</span>
						<span>Subscriptions, Project Scope & Counter Details</span>
					</div>
					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-sub-start">Subscription / Service Start</label>
							<input id="dtl-sub-start" type="date" class="form-input" bind:value={extraDetails.subscriptionPeriodStart} />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-sub-end">Subscription / Service End</label>
							<input id="dtl-sub-end" type="date" class="form-input" bind:value={extraDetails.subscriptionPeriodEnd} />
						</div>
						<div class="form-group" style="grid-column: 1 / -1; margin-bottom: 0;">
							<label class="form-label" for="dtl-project">Engagement / Project Name (Template 6 & Consulting)</label>
							<input id="dtl-project" type="text" class="form-input" bind:value={extraDetails.projectName} placeholder="e.g. Q3 Brand Identity Redesign" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-cashier">Cashier Name (Retail & POS)</label>
							<input id="dtl-cashier" type="text" class="form-input" bind:value={extraDetails.cashierName} placeholder="Cashier 01" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-counter">Counter Number (Retail & POS)</label>
							<input id="dtl-counter" type="text" class="form-input" bind:value={extraDetails.counterNumber} placeholder="Counter B" />
						</div>
					</div>
				</div>

				<hr style="border: 0; border-top: 1px solid var(--border-subtle); margin: 0;" />

				<!-- Statutory Terms & Policies -->
				<div>
					<div style="font-size: 0.8125rem; font-weight: 700; color: #a5b4fc; text-transform: uppercase; margin-bottom: 0.5rem; display: flex; align-items: center; gap: 0.35rem;">
						<span>📜</span>
						<span>Statutory Terms & Reverse Charge</span>
					</div>
					<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-terms">Payment Terms</label>
							<select id="dtl-terms" class="form-select" bind:value={extraDetails.terms}>
								<option value="Cash">Cash</option>
								<option value="Credit / Due">Credit / Due</option>
								<option value="Bank Transfer">Bank Transfer (NEFT/RTGS)</option>
								<option value="Cheque">Cheque</option>
								<option value="UPI / QR">UPI / QR</option>
								<option value="Net 15 Days">Net 15 Days</option>
								<option value="Net 30 Days">Net 30 Days</option>
							</select>
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-juris">Legal Jurisdiction City</label>
							<input id="dtl-juris" type="text" class="form-input" bind:value={extraDetails.jurisdictionCity} placeholder="Pune or Tirunelveli" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-interest">Interest Rate on Due Date</label>
							<input id="dtl-interest" type="text" class="form-input" bind:value={extraDetails.interestRate} placeholder="24%" />
						</div>
						<div class="form-group" style="margin-bottom: 0;">
							<label class="form-label" for="dtl-policy">Goods Return Policy</label>
							<input id="dtl-policy" type="text" class="form-input" bind:value={extraDetails.returnPolicy} placeholder="Goods once sold will not be taken back." />
						</div>
						<div class="form-group" style="grid-column: 1 / -1; margin-bottom: 0;">
							<label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.8125rem; cursor: pointer; color: var(--text-secondary);">
								<input type="checkbox" bind:checked={extraDetails.reverseCharge} />
								<span style="font-weight: 600; color: #fbbf24;">Tax is Payable on Reverse Charge (Section 9(3) / 9(4) of CGST Act)</span>
							</label>
						</div>
					</div>
				</div>

				<hr style="border: 0; border-top: 1px solid var(--border-subtle); margin: 0;" />

				<!-- Visual Display Toggles -->
				<div>
					<div style="font-size: 0.8125rem; font-weight: 700; color: #a5b4fc; text-transform: uppercase; margin-bottom: 0.5rem; display: flex; align-items: center; gap: 0.35rem;">
						<span>👁️</span>
						<span>Visual Display Toggles</span>
					</div>
					<div style="display: flex; flex-direction: column; gap: 0.5rem;">
						<label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.8125rem; cursor: pointer;">
							<input type="checkbox" bind:checked={extraDetails.showBankDetails} />
							<span>Display Bank Wire / Remittance Coordinates Box</span>
						</label>
						<label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.8125rem; cursor: pointer;">
							<input type="checkbox" bind:checked={extraDetails.showUPIQR} />
							<span>Display Instant UPI Scan-to-Pay QR Code</span>
						</label>
					</div>
				</div>
			</div>
			<div class="modal-footer">
				<button type="button" onclick={() => (showExtraDetailsModal = false)} class="btn btn-secondary">
					Close
				</button>
				<button type="button" onclick={saveExtraDetails} class="btn btn-primary">
					✓ Save & Update Invoice
				</button>
			</div>
		</div>
	</div>
{/if}
