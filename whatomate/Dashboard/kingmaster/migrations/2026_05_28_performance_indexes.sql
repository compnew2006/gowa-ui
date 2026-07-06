-- Performance indexes for frequent API filters, joins, and sort patterns.
-- Review with EXPLAIN on production-like data before applying.

CREATE INDEX idx_users_user_id ON users (user_id);
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_created_at ON users (created_at);
CREATE INDEX idx_users_timezone ON users (timezone);

CREATE INDEX idx_orders_status_created_at ON orders (status, created_at);
CREATE INDEX idx_orders_order_number ON orders (order_number);
CREATE INDEX idx_orders_tracking_code ON orders (tracking_code);
CREATE INDEX idx_order_status_history_order_id_created_at ON order_status_history (order_id, created_at);

CREATE INDEX idx_campaigns_user_type_created ON campaigns (user_id, type_tools, created_at);
CREATE INDEX idx_campaigns_user_updated ON campaigns (user_id, updated_at);
CREATE INDEX idx_campaigns_campaign_uid ON campaigns (campaign_uid);

CREATE INDEX idx_accounts_user_channel_created ON accounts (user_id, channel, created_at);
CREATE INDEX idx_accounts_user_updated ON accounts (user_id, updated_at);

CREATE INDEX idx_contacts_user_platform_created ON contacts (user_id, platform, created_at);
CREATE INDEX idx_content_user_created ON content (user_id, created_at);
CREATE INDEX idx_files_user_created ON files (user_id, created_at);

CREATE INDEX idx_notifications_user_read_created ON notifications (user_id, is_read, created_at);
CREATE INDEX idx_transactions_user_created ON transactions (user_id, created_at);
CREATE INDEX idx_wallet_transactions_sender_created ON wallet_transactions (sender_id, created_at);
CREATE INDEX idx_wallet_transactions_receiver_created ON wallet_transactions (receiver_id, created_at);

CREATE INDEX idx_user_subscriptions_user_end_status ON user_subscriptions (user_id, end_date, status);
CREATE INDEX idx_mlm_referrals_referrer ON mlm_referrals (referrer_id);
CREATE INDEX idx_mlm_referrals_user ON mlm_referrals (user_id);
