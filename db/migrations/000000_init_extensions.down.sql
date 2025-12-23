-- Drop update_updated_at_column function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop extensions (be careful with this in production)
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";
