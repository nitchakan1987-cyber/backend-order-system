# Orders API

## ตั้งค่า

ตั้งค่า `.env` เมื่อต้องรัน API จากเครื่อง:

```env
DATABASE_URL=myuser:mypass123@tcp(127.0.0.1:3306)/mydb?parseTime=true
HTTP_ADDR=:8080
```

ฐานข้อมูลใหม่: เริ่ม MySQL แล้วรัน schema `001` หนึ่งครั้ง

```powershell
docker compose up -d mysql
Get-Content -Raw database/migrations/001_api_schema.sql | docker compose exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot "$MYSQL_DATABASE"'
go run .
```

ฐานเดิม: สำรองข้อมูลก่อน แล้วรัน migration `002` หนึ่งครั้ง

```powershell
Get-Content -Raw database/migrations/002_align_database_to_001.sql | docker compose exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot mydb'
```

## สิทธิ์เรียก API

ทุก endpoint ใต้ `/api/v1` ต้องส่ง `Authorization: Bearer <token>` ระบบ login และออก token อยู่นอกขอบเขตนี้ เก็บ SHA-256 hash ของ token ใน `api_tokens` แล้วกำหนดผู้ขายที่ token เข้าถึงได้:

```sql
INSERT INTO api_tokens (token_hash) VALUES (LOWER(SHA2('replace-with-a-long-random-token', 256)));
SET @token_id = LAST_INSERT_ID();
INSERT INTO api_token_salespersons (token_id, salesperson_id) VALUES (@token_id, 1);
```

ผูกลูกค้ากับผู้ขายใน `salesperson_customers` ก่อนเรียก API ลูกค้าหรือสร้าง order

## Endpoints

| Method | Path | รายละเอียด |
| --- | --- | --- |
| GET | `/api/v1/salespersons` | ผู้ขายที่ token เข้าถึงได้ |
| GET | `/api/v1/customers?salespersonId=1` | ลูกค้าที่ผูกกับผู้ขาย |
| GET | `/api/v1/products` | รายการสินค้า |
| GET | `/api/v1/products/{productId}/price` | ราคาจาก `products.price` |
| GET | `/api/v1/orders?startDate=YYYY-MM-DD&endDate=YYYY-MM-DD&page=1&pageSize=50` | รายการ order และยอดรวม |
| GET | `/api/v1/orders/{orderId}` | รายละเอียด order |
| POST | `/api/v1/orders` | สร้าง order โดยใช้ราคาจากฐานข้อมูล |
| PUT | `/api/v1/orders/{orderId}` | แก้ order ที่ยังไม่ส่ง |
| DELETE | `/api/v1/orders/{orderId}` | soft delete order ที่ยังไม่ส่ง |
| GET | `/api/v1/reports/delivery-schedule?deliveryFrom=YYYY-MM-DD&deliveryTo=YYYY-MM-DD` | ตารางกำหนดส่ง |
| GET | `/api/v1/reports/delivery-schedule/export?deliveryFrom=YYYY-MM-DD&deliveryTo=YYYY-MM-DD&format=pdf` | ดาวน์โหลด PDF หรือ XLSX |

ยอดรวมและสถานะส่งคำนวณจาก `order_items`; สินค้าหนึ่งชนิดอยู่ใน order ได้หนึ่งรายการตาม primary key `(order_id, product_id)` หาก export PDF แล้วฟอนต์ไทยไม่แสดง ให้ตั้ง `PDF_FONT_PATH` เป็นไฟล์ Unicode TTF
