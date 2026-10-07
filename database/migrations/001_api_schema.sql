USE mydb;

CREATE TABLE salesperson (
    id MEDIUMINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(10),
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100),
    phone VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    update_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE api_tokens (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    token_hash CHAR(64) NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE api_token_salespersons (
    token_id BIGINT UNSIGNED NOT NULL,
    salesperson_id MEDIUMINT UNSIGNED NOT NULL,
    PRIMARY KEY (token_id, salesperson_id),
    FOREIGN KEY (token_id) REFERENCES api_tokens(id),
    FOREIGN KEY (salesperson_id) REFERENCES salesperson(id)
);

CREATE TABLE customers (
    id MEDIUMINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(10),
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100),
    phone VARCHAR(20),
    address VARCHAR(255),
    marketType VARCHAR(100) DEFAULT "NON_EXPORT",
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    update_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE salesperson_customers (
    salesperson_id MEDIUMINT UNSIGNED NOT NULL,
    customer_id MEDIUMINT UNSIGNED NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (salesperson_id, customer_id),
    FOREIGN KEY (salesperson_id) REFERENCES salesperson(id),
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);

CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    product_code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(12,2) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    update_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_no VARCHAR(50) NOT NULL UNIQUE,

    salesperson_id MEDIUMINT UNSIGNED NOT NULL,
    customer_id MEDIUMINT UNSIGNED NOT NULL,
	
    order_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    freight_charge DECIMAL(12,2) DEFAULT 0.00,
    insurance_charge DECIMAL(12,2) DEFAULT 0.00,
    update_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    version_update SMALLINT DEFAULT 1,
    deleted_at DATETIME NULL,
    FOREIGN KEY (salesperson_id)
        REFERENCES salesperson(id),

    FOREIGN KEY (customer_id)
        REFERENCES customers(id),

    INDEX idx_orders_salesperson (salesperson_id,customer_id)
);


CREATE TABLE order_items (
    order_id INT NOT NULL,
    product_id INT NOT NULL,

    quantity INT NOT NULL DEFAULT 0,
    price DECIMAL(12,2) NOT NULL,

    delivery_at DATETIME NULL,
    deliveryStatus VARCHAR(100) DEFAULT "NOT_SHIPPED",
    update_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (order_id,product_id),
    FOREIGN KEY (order_id)
        REFERENCES orders(id),

    FOREIGN KEY (product_id)
        REFERENCES products(id),

    INDEX idx_order_items_order (order_id),
    INDEX idx_order_items_product (product_id),
    INDEX idx_order_items_delivery (delivery_at)
);


