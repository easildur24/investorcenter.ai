-- Migration 027: Create notes tables for internal brainstorming tool
-- Hierarchy: feature_groups -> features -> feature_notes (with sections: ui, backend, data, infra)

-- feature_groups: top-level containers
CREATE TABLE IF NOT EXISTS feature_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    notes TEXT DEFAULT '',
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- features: belong to a group
CREATE TABLE IF NOT EXISTS features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES feature_groups(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    notes TEXT DEFAULT '',
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- feature_notes: individual notes under a feature's section
CREATE TABLE IF NOT EXISTS feature_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feature_id UUID NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    section VARCHAR(20) NOT NULL CHECK (section IN ('ui', 'backend', 'data', 'infra')),
    title VARCHAR(500) NOT NULL DEFAULT 'Untitled',
    content TEXT DEFAULT '',
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_features_group_id ON features(group_id);
CREATE INDEX IF NOT EXISTS idx_feature_notes_feature_id ON feature_notes(feature_id);
CREATE INDEX IF NOT EXISTS idx_feature_notes_section ON feature_notes(section);

-- Update triggers
CREATE OR REPLACE FUNCTION update_feature_groups_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_feature_groups_updated_at
    BEFORE UPDATE ON feature_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_feature_groups_updated_at();

CREATE OR REPLACE FUNCTION update_features_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_features_updated_at
    BEFORE UPDATE ON features
    FOR EACH ROW
    EXECUTE FUNCTION update_features_updated_at();

CREATE OR REPLACE FUNCTION update_feature_notes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_feature_notes_updated_at
    BEFORE UPDATE ON feature_notes
    FOR EACH ROW
    EXECUTE FUNCTION update_feature_notes_updated_at();
