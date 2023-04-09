CREATE TABLE a(id INTEGER PRIMARY KEY);
CREATE TABLE b(id INTEGER PRIMARY KEY);

INSERT INTO b (id) VALUES (1);
INSERT INTO a (id) VALUES (1);

ALTER TABLE a ADD COLUMN val_a INTEGER NOT NULL REFERENCES b(id) DEFAULT 1;
ALTER TABLE b ADD COLUMN val_b INTEGER NOT NULL REFERENCES b(id) DEFAULT 1;
