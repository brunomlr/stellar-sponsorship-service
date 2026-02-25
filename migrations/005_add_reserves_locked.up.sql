-- Track how many base reserves a signed transaction locks in the sponsor account
ALTER TABLE transaction_logs
    ADD COLUMN reserves_locked INT;
