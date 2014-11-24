USE mail;
INSERT INTO domains (domain) VALUES ('domain.com');
INSERT INTO users (email, password) VALUES ('user@domain.com', ENCRYPT('changeme'));

