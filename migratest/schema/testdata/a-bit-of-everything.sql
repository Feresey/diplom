-- Создание DOMAIN
CREATE DOMAIN name_domain AS VARCHAR(100);

-- Создание COMPOSITE типа
CREATE TYPE custom_type AS (
  first_name name_domain,
  last_name name_domain
);

-- Создание ENUM
CREATE TYPE status AS ENUM ('active', 'inactive');

CREATE TABLE circles (
  c circle,
  -- Добавление ограничения EXCLUDE
  EXCLUDE USING gist (c WITH &&)
);

-- Создание таблицы roles
CREATE TABLE roles (
  id INTEGER PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Создание таблицы с PRIMARY KEY, UNIQUE, CHECK, FOREIGN KEY ограничениями
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name name_domain NOT NULL,
  email VARCHAR(100) UNIQUE NOT NULL,
  age INTEGER CHECK (age > 0),
  full_name custom_type,
  status status,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  role_id INTEGER REFERENCES roles(id),
  -- Добавление колонки с пользовательским типом, основанным на RANGE
  price_range INT4RANGE NOT NULL CONSTRAINT users_price_range_check CHECK (price_range <> '[,)'),
  -- Добавление CHECK условия для нескольких колонок
  CONSTRAINT users_check CHECK (
    (
      age > 18
      AND status = 'active'
    )
    OR (status = 'inactive')
  )
);

-- Создание таблицы с ARRAY и RANGE типами данных
CREATE TABLE products (
  id INTEGER PRIMARY KEY,
  name VARCHAR(50),
  prices INTEGER [] NOT NULL,
  discount_range INT4RANGE,
  quantity_range INT4RANGE NOT NULL CONSTRAINT products_quantity_range_check CHECK (quantity_range <> '[,)')
);

-- Создание связи многие ко многим
CREATE TABLE orders (
  id INTEGER PRIMARY KEY,
  user_id INTEGER REFERENCES users(id),
  product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
  quantity INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  -- Добавление UNIQUE индекса для нескольких колонок
  CONSTRAINT orders_user_product_unique UNIQUE NULLS NOT DISTINCT (user_id, product_id)
);

-- Добавление обычного индекса
CREATE INDEX orders_user_id_idx ON orders(user_id);

-- Создание таблицы с GENERATED COLUMN
CREATE TABLE employers (
  id INTEGER PRIMARY KEY,
  first_name VARCHAR(50) NOT NULL,
  last_name VARCHAR(50) NOT NULL,
  salary REAL NOT NULL,
  bonus_percent REAL NOT NULL,
  total_salary REAL GENERATED ALWAYS AS (salary + (salary * bonus_percent / 100)) STORED
);

-- Создание связи один ко многим
CREATE TABLE departments (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  manager_id INTEGER REFERENCES employers(id) UNIQUE
);

-- Создание триггера для автоматического обновления даты изменения записи в таблице roles
CREATE OR REPLACE FUNCTION update_roles_updated_at()
RETURNS TRIGGER AS
$$
BEGIN NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER roles_update_trigger
BEFORE UPDATE ON roles
  FOR EACH ROW EXECUTE FUNCTION update_roles_updated_at();

-- Создание триггера для автоматического добавления значения по умолчанию в колонку created_at таблицы employers
CREATE OR REPLACE FUNCTION set_employers_created_at()
RETURNS TRIGGER AS
$$
BEGIN NEW.created_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER employers_insert_trigger
BEFORE INSERT ON employers
  FOR EACH ROW EXECUTE FUNCTION set_employers_created_at();