-- Watch Lists table
CREATE TABLE IF NOT EXISTS watch_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    display_order INTEGER,
    is_public BOOLEAN DEFAULT FALSE, -- For future sharing feature
    public_slug VARCHAR(100) UNIQUE, -- For future public URLs
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for watch_lists
CREATE INDEX IF NOT EXISTS idx_watch_lists_user_id ON watch_lists(user_id);
CREATE INDEX IF NOT EXISTS idx_watch_lists_user_id_display_order ON watch_lists(user_id, display_order);

-- Watch List Items table
CREATE TABLE IF NOT EXISTS watch_list_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    watch_list_id UUID REFERENCES watch_lists(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL, -- References stocks.symbol
    notes TEXT,
    tags TEXT[], -- Array of custom tags
    target_buy_price DECIMAL(20, 4),
    target_sell_price DECIMAL(20, 4),
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    display_order INTEGER,
    UNIQUE(watch_list_id, symbol) -- Prevent duplicate tickers in same watch list
);

-- Indexes for watch_list_items
CREATE INDEX IF NOT EXISTS idx_watch_list_items_watch_list_id ON watch_list_items(watch_list_id);
CREATE INDEX IF NOT EXISTS idx_watch_list_items_symbol ON watch_list_items(symbol);
CREATE INDEX IF NOT EXISTS idx_watch_list_items_watch_list_id_display_order ON watch_list_items(watch_list_id, display_order);

-- Trigger to update updated_at timestamp for watch_lists
CREATE OR REPLACE FUNCTION update_watch_lists_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_watch_lists_updated_at ON watch_lists;
CREATE TRIGGER update_watch_lists_updated_at
BEFORE UPDATE ON watch_lists
FOR EACH ROW EXECUTE FUNCTION update_watch_lists_updated_at();

-- Function to enforce watch list limits for free tier
CREATE OR REPLACE FUNCTION check_watch_list_item_limit()
RETURNS TRIGGER AS $$
DECLARE
    user_is_premium BOOLEAN;
    current_item_count INTEGER;
    max_items INTEGER := 10; -- Free tier limit
BEGIN
    -- Get user's premium status
    SELECT u.is_premium INTO user_is_premium
    FROM watch_lists wl
    JOIN users u ON wl.user_id = u.id
    WHERE wl.id = NEW.watch_list_id;

    -- Skip check for premium users
    IF user_is_premium THEN
        RETURN NEW;
    END IF;

    -- Count existing items in watch list
    SELECT COUNT(*) INTO current_item_count
    FROM watch_list_items
    WHERE watch_list_id = NEW.watch_list_id;

    -- Check if limit exceeded
    IF current_item_count >= max_items THEN
        RAISE EXCEPTION 'Watch list limit reached. Free tier allows maximum % tickers per watch list.', max_items;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to enforce watch list item limit
DROP TRIGGER IF EXISTS enforce_watch_list_item_limit ON watch_list_items;
CREATE TRIGGER enforce_watch_list_item_limit
BEFORE INSERT ON watch_list_items
FOR EACH ROW EXECUTE FUNCTION check_watch_list_item_limit();

-- Function to create default watch list for new users
CREATE OR REPLACE FUNCTION create_default_watch_list()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO watch_lists (user_id, name, description, is_default, display_order)
    VALUES (NEW.id, 'My Watch List', 'Default watch list', TRUE, 0);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-create default watch list when user signs up
DROP TRIGGER IF EXISTS auto_create_default_watch_list ON users;
CREATE TRIGGER auto_create_default_watch_list
AFTER INSERT ON users
FOR EACH ROW EXECUTE FUNCTION create_default_watch_list();
