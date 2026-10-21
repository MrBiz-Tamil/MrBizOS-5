-- 014_ledger_balance_constraint.sql
-- Database-level double-entry invariant. Application code already rejects unbalanced vouchers; this
-- makes the database refuse them too, so no code path (including future ones or manual SQL) can commit
-- a journal entry whose lines do not balance and match the header totals.
--
-- Deferred constraint triggers run at COMMIT, after the header and all its lines were inserted in the
-- same transaction. Existing rows are not re-validated (only new inserts are checked).

CREATE OR REPLACE FUNCTION fn_assert_journal_balanced()
RETURNS TRIGGER AS $$
DECLARE
    eid UUID;
    line_dr NUMERIC;
    line_cr NUMERIC;
    hdr_dr NUMERIC;
    hdr_cr NUMERIC;
BEGIN
    IF TG_TABLE_NAME = 'journal_entries' THEN
        eid := NEW.id;
    ELSE
        eid := NEW.journal_entry_id;
    END IF;

    SELECT total_debit, total_credit INTO hdr_dr, hdr_cr FROM journal_entries WHERE id = eid;
    IF hdr_dr IS NULL THEN
        RETURN NULL; -- entry no longer exists
    END IF;

    SELECT COALESCE(SUM(debit_amount), 0), COALESCE(SUM(credit_amount), 0)
      INTO line_dr, line_cr
      FROM journal_lines WHERE journal_entry_id = eid;

    IF line_dr <> line_cr OR line_dr <> hdr_dr OR line_cr <> hdr_cr THEN
        RAISE EXCEPTION 'Unbalanced journal entry %: lines debit %, credit %; header debit %, credit %',
            eid, line_dr, line_cr, hdr_dr, hdr_cr
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_journal_entries_balanced ON journal_entries;
CREATE CONSTRAINT TRIGGER trg_journal_entries_balanced
AFTER INSERT ON journal_entries
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION fn_assert_journal_balanced();

DROP TRIGGER IF EXISTS trg_journal_lines_balanced ON journal_lines;
CREATE CONSTRAINT TRIGGER trg_journal_lines_balanced
AFTER INSERT ON journal_lines
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION fn_assert_journal_balanced();

