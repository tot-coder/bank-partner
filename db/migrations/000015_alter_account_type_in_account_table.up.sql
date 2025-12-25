ALTER TABLE accounts
ADD CONSTRAINT accounts_account_type_check 
CHECK (account_type IN ('SAVINGS', 'CHECKING', 'CURRENT'));
