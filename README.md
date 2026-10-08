# Orders API

Backend สำหรับจัดการคำสั่งซื้อ ลูกค้า สินค้า และรายงานกำหนดส่ง พัฒนาด้วย Go, Gin และ MySQL

## โครงสร้างโปรเจกต์

```text
.
├── api/                    # HTTP handlers, request validation และ API tests
├── app/                    # ประกอบ dependencies และเริ่ม HTTP server
├── config/                 # โหลดและตรวจสอบการตั้งค่า
├── database/               # การเชื่อมต่อ MySQL
│   └── migrations/         # SQL schema สำหรับเริ่มฐานข้อมูล
├── health/                 # Health check
├── repository/             # คำสั่งอ่านและเขียนข้อมูลใน MySQL
├── response/               # รูปแบบ response ของ health check
├── router/                 # ลงทะเบียน routes และ middleware
├── service/                # กฎทางธุรกิจและการสร้างรายงาน
├── compose.yaml            # MySQL สำหรับการพัฒนา
├── main.go                 # จุดเริ่มต้นของโปรแกรม
└── go.mod                  # Go module และ dependencies
```

## สิ่งที่ต้องมี

- Go เวอร์ชันที่ระบุใน `go.mod` หรือใหม่กว่า
- Docker Compose
- ไฟล์ `.env` ที่มีค่า `DATABASE_URL` และ `HTTP_ADDR`

ตัวอย่าง `.env` สำหรับใช้กับ MySQL ใน `compose.yaml`:

```env
DATABASE_URL=myuser:mypass123@tcp(127.0.0.1:3306)/mydb?parseTime=true
HTTP_ADDR=:8080
```

ปรับรหัสผ่านให้เหมาะสมก่อนนำไปใช้ในสภาพแวดล้อมอื่น

## เริ่มใช้งาน

รันคำสั่งจากโฟลเดอร์หลักของโปรเจกต์:

1. เริ่ม MySQL:

   ```powershell
   docker compose up -d mysql
   ```

2. สร้างตารางจาก schema เริ่มต้น (รันกับฐานข้อมูลใหม่ที่ยังไม่มีตาราง):

   ```powershell
   Get-Content -Raw database/migrations/001_api_schema.sql | docker compose exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot "$MYSQL_DATABASE"'
   ```

   ปัจจุบันมีไฟล์ schema เริ่มต้น `database/migrations/001_api_schema.sql` เพียงไฟล์เดียว หากใช้ฐานข้อมูลเดิม ให้สำรองข้อมูลและตรวจสอบ schema ก่อนนำ SQL นี้ไปรัน

3. เริ่ม API:

   ```powershell
   go run .
   ```

เซิร์ฟเวอร์ใช้ค่าจาก `HTTP_ADDR` (ค่าเริ่มต้น `:8080`) และต้องเชื่อมต่อ MySQL ได้ก่อนเริ่มทำงาน

## การเตรียมสิทธิ์เรียก API

ทุก endpoint ใต้ `/api/v1` ต้องส่ง access token ผ่าน header `Authorization` โดยใช้ `Bearer` scheme ระบบ login และออก token ไม่ได้รวมอยู่ใน backend นี้ ให้สร้าง token ที่คาดเดายาก เก็บเฉพาะ SHA-256 hash ใน `api_tokens` และกำหนดผู้ขายที่ token เข้าถึงได้:

```sql
INSERT INTO api_tokens (token_hash)
VALUES (LOWER(SHA2('replace-with-a-long-random-token', 256)));

SET @token_id = LAST_INSERT_ID();

INSERT INTO api_token_salespersons (token_id, salesperson_id)
VALUES (@token_id, 1);
```

ส่งค่า token ต้นฉบับใน header เมื่อเรียก API อย่าส่ง hash แทน token และตรวจให้แน่ใจว่ามีข้อมูลผู้ขายอยู่จริง ก่อนเรียก API ที่ทำงานกับลูกค้า ให้ผูกลูกค้ากับผู้ขายใน `salesperson_customers` ด้วย

## Endpoints

`GET /health` ใช้ตรวจสอบสถานะ API และการเชื่อมต่อฐานข้อมูล โดยไม่ต้องใช้ token ส่วน endpoints อื่นทั้งหมดต้องยืนยันตัวตน

| Method | Path | รายละเอียด |
| --- | --- | --- |
| GET | `/health` | ตรวจสอบสถานะ API และ MySQL |
| GET | `/api/v1/salespersons` | รายชื่อผู้ขายที่ token เข้าถึงได้ |
| GET | `/api/v1/customers?salespersonId=1` | รายชื่อลูกค้า เลือกกรองตามผู้ขายได้ |
| GET | `/api/v1/customers/all` | รายชื่อลูกค้าทั้งหมด |
| GET | `/api/v1/products` | รายการสินค้า |
| GET | `/api/v1/products/{productId}/price` | ราคาสินค้าจาก `products.price` |
| GET | `/api/v1/orders?startDate=YYYY-MM-DD&endDate=YYYY-MM-DD&page=1&pageSize=50` | รายการคำสั่งซื้อและสรุปยอด; วันที่ต้องระบุ ส่วน `page` และ `pageSize` ไม่บังคับ |
| GET | `/api/v1/orders/{orderId}` | รายละเอียดคำสั่งซื้อ |
| POST | `/api/v1/orders` | สร้างคำสั่งซื้อ โดยใช้ราคาสินค้าจากฐานข้อมูล |
| PUT | `/api/v1/orders/{orderId}` | แก้ไขคำสั่งซื้อที่ยังไม่ส่ง |
| DELETE | `/api/v1/orders/{orderId}` | ลบคำสั่งซื้อที่ยังไม่ส่งแบบ soft delete |
| GET | `/api/v1/reports/delivery-schedule?deliveryFrom=YYYY-MM-DD&deliveryTo=YYYY-MM-DD` | ตารางกำหนดส่งสินค้าในช่วงวันที่ |
| GET | `/api/v1/reports/delivery-schedule/export?deliveryFrom=YYYY-MM-DD&deliveryTo=YYYY-MM-DD&format=pdf` | ดาวน์โหลดรายงานเป็น PDF หรือ XLSX (`format=pdf` หรือ `format=xlsx`) |

`page` เริ่มต้นที่ `1` และ `pageSize` เริ่มต้นที่ `50` โดยกำหนด `pageSize` ได้ตั้งแต่ `1` ถึง `100` ยอดรวมและสถานะการส่งคำนวณจาก `order_items`; สินค้าหนึ่งชนิดอยู่ในคำสั่งซื้อเดียวกันได้หนึ่งรายการ ตาม primary key `(order_id, product_id)` หาก PDF แสดงฟอนต์ไทยไม่ถูกต้อง ให้กำหนด `PDF_FONT_PATH` เป็น path ของไฟล์ Unicode TTF

## ทดสอบ

```powershell
go test ./...
```
