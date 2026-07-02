# بخش سوم تمرین — ساخت سرویس، کانتینری‌سازی و اجرا در ماشین مجازی

## توضیح پروژه

در این بخش، یک سرویس محاسباتی ساده به زبان Go پیاده‌سازی شده است که از طریق پروتکل HTTP درخواست‌ها را دریافت می‌کند. این سرویس در قالب یک Docker image ساخته شده و داخل یک ماشین مجازی اجرا می‌شود. هدف اصلی این آزمایش، آشنایی با تبدیل برنامه به سرویس شبکه‌ای، کانتینری‌سازی، و برقراری ارتباط از سیستم Host به سرویس داخل Guest VM است.

سرویس دو endpoint اصلی دارد: یکی برای بررسی سلامت (`/health`) و دیگری برای انجام عملیات‌های محاسباتی (`/compute`). تمام خطاهای مورد انتظار مانند تقسیم بر صفر، عمل نامعتبر، پارامترهای missing، و ورودی غیرعددی به درستی مدیریت می‌شوند.

---

# ساختار فایل‌ها
```text
part3/
├── main.go
├── Dockerfile
├── README.md
```

---

# پیش‌نیازها

* Linux Operating System
* GO Language
* Docker
* curl (برای تست)

---

# تنظیم GOPROXY

مطابق صورت تمرین، قبل از اجرا دستور زیر را اجرا کنید:
```text
go env -w GOPROXY=https://mirror-go.runflare.com
```

---

# نحوه ساخت Docker Image

ابتدا وارد پوشه `part3` شوید:
```text
cd part3
```

سپس image را بسازید:
```text
sudo docker build -t hw1-part3 .
```

خروجی موفقیت‌آمیز:
```text
Successfully tagged hw1-part3:latest
```

---

# نحوه اجرای کانتینر
```text
sudo docker run -p 8080:8080 hw1-part3
```


خروجی موفقیت‌آمیز:
```text
[server] Starting server on :8080
```


نکته: سرویس روی پورت `8080` در دسترس خواهد بود.

---

# تست سرویس

در یک ترمینال جدید (یا از روی Host) دستورات زیر را اجرا کنید.

---

## 1. Health Check
```text
curl http://localhost:8080/health
```


خروجی موفقیت‌آمیز:
```text
OK
```


---

## 2. عملیات ADD
```text
curl "http://localhost:8080/compute?op=ADD&a=5&b=7"
```


خروجی موفقیت‌آمیز:
```JSON
{"status":"ok","operation":"ADD","a":5,"b":7,"result":12}
```


---

## 3. عملیات SUB
```text
curl "http://localhost:8080/compute?op=SUB&a=10&b=3"
```


خروجی موفقیت‌آمیز:
```JSON
{"status":"ok","operation":"SUB","a":10,"b":3,"result":7}
```


---

## 4. عملیات MUL
```text
curl "http://localhost:8080/compute?op=MUL&a=4&b=6"
```


خروجی موفقیت‌آمیز:
```JSON
{"status":"ok","operation":"MUL","a":4,"b":6,"result":24}
```


---

## 5. عملیات DIV
```text
curl "http://localhost:8080/compute?op=DIV&a=8&b=2"
```


خروجی موفقیت‌آمیز:
```JSON
{"status":"ok","operation":"DIV","a":8,"b":2,"result":4}
```


---

# مدیریت خطاها

## تقسیم بر صفر
```text
curl "http://localhost:8080/compute?op=DIV&a=5&b=0"
```


خروجی:
```JSON
{"status":"err","error":"division_by_zero"}
```


---

## عمل نامعتبر
```text
curl "http://localhost:8080/compute?op=POW&a=2&b=3"
```


خروجی:
```JSON
{"status":"err","error":"unknown_operation"}
```


---

## پارامتر missing
```text
curl "http://localhost:8080/compute?op=ADD&a=5"
```


خروجی:
```text
missing parameters
```


---

## ورودی غیرعددی
```text
curl "http://localhost:8080/compute?op=ADD&a=abc&b=5"
```


خروجی:
```test
invalid a
```


---

# بررسی کانتینر در حال اجرا
```text
sudo docker ps
```


خروجی نمونه:
```text
CONTAINER ID IMAGE COMMAND CREATED STATUS PORTS NAMES
4bc5bc05b0f8 hw1-part3 "./server" 8 minutes ago Up 8 minutes 0.0.0.0:8080->8080/tcp, [::]:8080->8080/tcp silly_nash
```


---

# توقف کانتینر

با `Ctrl+C` در ترمینال اجرا، یا دستور زیر:
```text
sudo docker stop <container_id>
```


---

# اجرای مجدد بعداً
```text
cd part3
sudo docker run -p 8080:8080 hw1-part3
```


---

# بازسازی بعد از تغییر کد
```text
docker build -t hw1-part3 .
docker run -p 8080:8080 hw1-part3
```


---

# تست از روی Windows Host (PowerShell)

## Health Check
```text
curl http://localhost:8080/health
```


خروجی:
```text
OK
```


## عملیات محاسباتی
```text
curl "http://localhost:8080/compute?op=SUB&a=10&b=3"
```


خروجی:
```JSON
{"status":"ok","operation":"SUB","a":10,"b":3,"result":7}
```


---

# خلاصه پورت‌ها و دسترسی

| آیتم | مقدار |
|------|-------|
| پورت سرویس | 8080 |
| پورت منتشر شده روی Host | 8080 |
| آدرس دسترسی از Host | http://localhost:8080 |
| آدرس دسترسی از VM (داخل کانتینر) | :8080 |

---

# اثبات اجرا در ماشین مجازی

برای اثبات اجرای موفق سرویس داخل ماشین مجازی و اتصال از Host:

1. کانتینر داخل VM اجرا می‌شود
2. پورت 8080 از VM به Host forward شده است
3. درخواست‌های curl از Host به موفقیت پاسخ دریافت می‌کنند
4. لاگ‌های سرویس در ترمینال VM نمایش داده می‌شود.
