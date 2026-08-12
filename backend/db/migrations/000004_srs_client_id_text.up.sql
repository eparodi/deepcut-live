-- SRS 5 sends client_id in http_hooks callbacks as a string connection id
-- (e.g. "5u9c4d30"), not a number. Widen the column accordingly.
ALTER TABLE streams ALTER COLUMN srs_client_id TYPE TEXT
    USING (CASE WHEN srs_client_id IS NULL THEN NULL ELSE srs_client_id::TEXT END);
