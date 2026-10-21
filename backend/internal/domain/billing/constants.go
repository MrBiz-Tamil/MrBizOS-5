package billing

import "drayya-core/internal/domain/finance"

// PlatformCommissionPercent re-exports the canonical commission rate from the
// finance package so billing code reads naturally without risking drift. The
// single source of truth lives in finance.ComputeConsultationSplit.
const PlatformCommissionPercent = finance.PlatformCommissionPercent
