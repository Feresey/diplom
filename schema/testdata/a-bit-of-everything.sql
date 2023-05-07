CREATE SCHEMA IF NOT EXISTS test;

-- Создание DOMAIN
CREATE DOMAIN test.name_domain AS VARCHAR(100);

-- Создание COMPOSITE типа
CREATE TYPE test.custom_type AS (
  first_name test.name_domain,
  last_name test.name_domain
);

-- Создание ENUM
CREATE TYPE test.status AS ENUM ('active', 'inactive');

CREATE TABLE test.circles (
  c circle,
  -- Добавление ограничения EXCLUDE
  EXCLUDE USING gist (c WITH &&)
);

-- Создание таблицы roles
CREATE TABLE test.roles (
  id INTEGER PRIMARY KEY,
  name VARCHAR(50) UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE (id, name)
);

-- Создание таблицы с PRIMARY KEY, UNIQUE, CHECK, FOREIGN KEY ограничениями
CREATE TABLE test.users (
  id SERIAL PRIMARY KEY,
  name test.name_domain NOT NULL,
  email VARCHAR(100) UNIQUE NOT NULL,
  age INTEGER CHECK (age > 0),
  full_name test.custom_type,
  status test.status,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  role_id INTEGER NOT NULL,
  role_name VARCHAR(50) NOT NULL,
  CONSTRAINT user_role_composite_fk FOREIGN KEY (role_id, role_name) REFERENCES test.roles(id, name),
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
CREATE TABLE test.products (
  id INTEGER PRIMARY KEY,
  name VARCHAR(50),
  prices_1 INTEGER [] NOT NULL,
  prices_2 INTEGER [2][3] NOT NULL,
  prices_3 INTEGER [][] NOT NULL,
  discount_range INT4RANGE,
  quantity_range INT4RANGE NOT NULL CONSTRAINT products_quantity_range_check CHECK (quantity_range <> '[,)')
);

-- Создание связи многие ко многим
CREATE TABLE test.orders (
  id INTEGER PRIMARY KEY,
  user_id INTEGER REFERENCES test.users(id),
  product_id INTEGER REFERENCES test.products(id) ON DELETE CASCADE,
  quantity INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  -- Добавление UNIQUE индекса для нескольких колонок
  CONSTRAINT orders_user_product_unique UNIQUE NULLS NOT DISTINCT (user_id, product_id)
);

-- Добавление обычного индекса
CREATE INDEX test_orders_user_id_idx ON test.orders(user_id);

-- Создание таблицы с GENERATED COLUMN
CREATE TABLE test.employers (
  id INTEGER PRIMARY KEY,
  first_name VARCHAR(50) NOT NULL,
  last_name VARCHAR(50) NOT NULL,
  salary REAL NOT NULL,
  bonus_percent REAL NOT NULL,
  total_salary REAL GENERATED ALWAYS AS (salary + (salary * bonus_percent / 100)) STORED
);

-- Создание связи один ко многим
CREATE TABLE test.departments (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  manager_id INTEGER REFERENCES test.employers(id) UNIQUE
);

-- Создание триггера для автоматического обновления даты изменения записи в таблице roles
CREATE OR REPLACE FUNCTION test.update_roles_updated_at()
RETURNS TRIGGER AS
$$
BEGIN NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER roles_update_trigger
BEFORE UPDATE ON test.roles
  FOR EACH ROW EXECUTE FUNCTION test.update_roles_updated_at();

-- Создание триггера для автоматического добавления значения по умолчанию в колонку created_at таблицы employers
CREATE OR REPLACE FUNCTION test.set_employers_created_at()
RETURNS TRIGGER AS
$$
BEGIN NEW.created_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER employers_insert_trigger
BEFORE INSERT ON test.employers
  FOR EACH ROW EXECUTE FUNCTION test.set_employers_created_at();