````markdown
# Part 1 — Named Pipe Calculator (IPC with FIFO)

این بخش شامل پیاده‌سازی یک ماشین‌حساب ساده با استفاده از ارتباط بین دو پردازه
(Inter-Process Communication)
در زبان Go است.

ارتباط بین برنامه‌ها با استفاده از
Named Pipe (FIFO)
انجام می‌شود.

دو برنامه مستقل وجود دارند:

- `worker.go`
- `interface.go`

برنامه worker نقش سرور محاسباتی را دارد و برنامه interface ورودی کاربر را دریافت می‌کند.

---

# ساختار پروژه

```text
part1/
├── interface.go
├── worker.go
└── README.md
```

---

# توضیح معماری برنامه

در این پروژه دو FIFO ایجاد می‌شود:

| Pipe | کاربرد |
|---|---|
| `/tmp/hw1_part1_req_write` | ارسال درخواست از interface به worker |
| `/tmp/hw1_part1_res_write` | ارسال پاسخ از worker به interface |

---

# نحوه عملکرد

## مرحله 1 — اجرای Worker

برنامه worker ابتدا دو pipe را ایجاد می‌کند و منتظر اتصال interface می‌ماند.

سپس:

- درخواست‌ها را دریافت می‌کند
- عملیات محاسباتی را انجام می‌دهد
- نتیجه را به interface برمی‌گرداند

---

## مرحله 2 — اجرای Interface

برنامه interface:

- به pipeها متصل می‌شود
- ورودی کاربر را دریافت می‌کند
- درخواست را به worker ارسال می‌کند
- پاسخ را نمایش می‌دهد

---

# عملیات‌های پشتیبانی‌شده

```text
ADD
SUB
MUL
DIV
```

فرمت ورودی:

```text
OP A B
```

مثال:

```text
ADD 5 3
```

---

# نحوه اجرا

## مرحله 1 — اجرای Worker

در ترمینال اول:

```bash
cd part1
go run worker.go
```

خروجی مورد انتظار:

```text
[worker] Pipes created. Waiting for interface to connect...
```

---

## مرحله 2 — اجرای Interface

در ترمینال دوم:

```bash
cd part1
go run interface.go
```

خروجی مورد انتظار:

```text
[interface] Connecting to worker...
[interface] Connected to worker!
[interface] Connected! Enter operations as: OP A B  (e.g. ADD 5 3)
[interface] Supported: ADD, SUB, MUL, DIV — type 'exit' to quit.
```

و در ترمینال worker:

```text
[worker] Interface connected!
[worker] Ready. Waiting for requests...
```

---

# تست عملیات‌ها

---

## تست ADD

ورودی در interface:

```text
ADD 5 3
```

خروجی interface:

```text
[interface] Result: 8
```

خروجی worker:

```text
2026/05/14 12:00:00 [worker] Received: op=ADD a=5 b=3
2026/05/14 12:00:00 [worker] Sent: {"status":"ok","result":8}
```

---

## تست SUB

ورودی:

```text
SUB 10 4
```

خروجی interface:

```text
[interface] Result: 6
```

خروجی worker:

```text
2026/05/14 12:00:05 [worker] Received: op=SUB a=10 b=4
2026/05/14 12:00:05 [worker] Sent: {"status":"ok","result":6}
```

---

## تست MUL

ورودی:

```text
MUL 7 6
```

خروجی interface:

```text
[interface] Result: 42
```

خروجی worker:

```text
2026/05/14 12:00:10 [worker] Received: op=MUL a=7 b=6
2026/05/14 12:00:10 [worker] Sent: {"status":"ok","result":42}
```

---

## تست DIV

ورودی:

```text
DIV 20 5
```

خروجی interface:

```text
[interface] Result: 4
```

خروجی worker:

```text
2026/05/14 12:00:15 [worker] Received: op=DIV a=20 b=5
2026/05/14 12:00:15 [worker] Sent: {"status":"ok","result":4}
```

---

# تست خطاها

---

## تقسیم بر صفر

ورودی:

```text
DIV 5 0
```

خروجی interface:

```text
[interface] Error from worker: division_by_zero
```

خروجی worker:

```text
2026/05/14 12:00:20 [worker] Received: op=DIV a=5 b=0
2026/05/14 12:00:20 [worker] Sent: {"status":"err","error":"division_by_zero"}
```

---

## عملیات نامعتبر

ورودی:

```text
POW 2 3
```

خروجی interface:

```text
[interface] Error: unknown operation 'POW'. Use ADD SUB MUL DIV
```

---

## ورودی نامعتبر

ورودی:

```text
ADD x 5
```

خروجی:

```text
[interface] Error: 'x' is not a valid integer
```

---

## فرمت اشتباه

ورودی:

```text
ADD 5
```

خروجی:

```text
[interface] Error: expected format is OP A B (e.g. ADD 5 3)
```

---

# خروج از برنامه

برای خروج:

```text
exit
```

خروجی interface:

```text
[interface] Exiting.
```

خروجی worker:

```text
[worker] Interface disconnected. Waiting for a new connection...
[worker] Restarting pipes...
[worker] Pipes created. Waiting for interface to connect...
```

---

# ویژگی‌های پیاده‌سازی

در این پروژه ویژگی‌های زیر پیاده‌سازی شده‌اند:

- استفاده از Named Pipe (FIFO)
- ارتباط دوطرفه بین پردازه‌ها
- استفاده از JSON برای تبادل داده
- مدیریت خطاها
- مدیریت قطع اتصال
- مدیریت Broken Pipe
- امکان اتصال مجدد interface
- اعتبارسنجی ورودی کاربر
- لاگ‌گیری عملیات‌ها

---

# ساختار پیام‌ها

## Request

```json
{
  "operation": "ADD",
  "operand1": 5,
  "operand2": 3
}
```

---

## Response

### موفق

```json
{
  "status": "ok",
  "result": 8
}
```

### خطا

```json
{
  "status": "err",
  "error": "division_by_zero"
}
```

---

# پاک کردن Pipeها در صورت نیاز

اگر pipeها باقی مانده باشند:

```bash
rm -f /tmp/hw1_part1_req_write
rm -f /tmp/hw1_part1_res_write
```

---

# تست مجدد

پس از بستن برنامه‌ها:

```bash
go run worker.go
```

و سپس:

```bash
go run interface.go
```

---

# نتیجه‌گیری

در این بخش یک سیستم IPC ساده با استفاده از Named Pipe در Go طراحی و پیاده‌سازی شد.

برنامه شامل:

- ارتباط همزمان بین دو پردازه
- تبادل پیام JSON
- مدیریت خطا
- تحمل قطع اتصال
- طراحی Client/Worker

بود و رفتار سیستم در شرایط مختلف بررسی شد.
````
