# بخش دوم تمرین — همزمانی، زمان‌بندی و مشاهده اثرات Context Switching

## توضیح پروژه

در این بخش، برنامه‌ای به زبان Go پیاده‌سازی شده است که رفتار اجرای همزمان تعداد زیادی واحد اجرایی را تحت شرایط مختلف بررسی می‌کند. هدف اصلی این آزمایش، مشاهده اثرات افزایش همزمانی، زمان‌بندی توسط Go Scheduler، و تاثیر مقدار `GOMAXPROCS` بر عملکرد سیستم است.

برنامه با اجرای workloadهای مختلف و تغییر تعداد goroutineها و مقدار `GOMAXPROCS`، متریک‌های عملکردی مختلف را اندازه‌گیری کرده و نتایج را در قالب فایل CSV ذخیره می‌کند تا امکان تحلیل و رسم نمودار فراهم شود.

---

# ساختار فایل‌ها

```text
part2/
├── main.go
├── README.md
├── results/
│   ├── benchmark.csv
│   ├── throughput_cpu.png
│   ├── throughput_io.png
│   ├── throughput_mixed.png
│   ├── time_cpu.png
│   ├── latency_cpu.png
│   └── ...
└── trace.out
````

---

# پیش‌نیازها

* Linux Operating System
* GO Language
* Python3 
* Libraries:

  * pandas
  * matplotlib

---

# تنظیم GOPROXY

مطابق صورت تمرین، قبل از اجرا دستور زیر را اجرا کنید:

```bash
go env -w GOPROXY=https://mirror-go.runflare.com
```

---

# نحوه اجرای برنامه

ابتدا وارد پوشه `part2` شوید:

```bash
cd part2
```

سپس برنامه را اجرا کنید:

```bash
go run main.go
```

پس از پایان اجرا:

* فایل نتایج در مسیر زیر ذخیره می‌شود:

```text
results/benchmark.csv
```

* فایل trace نیز تولید می‌شود:

```text
trace.out
```

---

# توضیح workloadها

برنامه شامل سه نوع workload است:

## 1. CPU-bound

در این workload عملیات محاسباتی سنگین (محاسبه اعداد اول) انجام می‌شود تا فشار روی CPU و Scheduler بررسی شود.

---

## 2. IO-bound

در این workload از `time.Sleep` برای شبیه‌سازی عملیات blocking و IO استفاده شده است.

---

## 3. Mixed

ترکیبی از workload محاسباتی و blocking است.

---

# پارامترهای آزمایش

در این پروژه سه پارامتر اصلی تغییر داده می‌شوند:

## تعداد goroutineها

```text
1, 2, 4, 8, 16, 32, 64, 128
```

---

## مقدار GOMAXPROCS

```text
1,
2,
runtime.NumCPU()
```

---

## نوع workload

```text
cpu
io
mixed
```

---

# متریک‌های اندازه‌گیری‌شده

برای هر اجرای benchmark متریک‌های زیر ثبت می‌شوند:

* زمان کل اجرا
* throughput
* میانگین latency
* حداقل latency
* حداکثر latency
* انحراف معیار latency
* تعداد goroutineها
* مقدار GOMAXPROCS

---

# فرمت فایل benchmark.csv

فایل خروجی شامل ستون‌های زیر است:

```text
workload,
goroutines,
gomaxprocs,
total_time_ms,
throughput,
avg_latency_us,
min_latency_us,
max_latency_us,
stddev_latency_us
```

---

# تولید نمودارها

برای رسم نمودارها از Python استفاده شده است.

ابتدا کتابخانه‌های زیر را نصب کنید:

```bash
pip install pandas matplotlib
```

سپس فایل رسم نمودار را اجرا کنید:

```bash
python3 plot_results.py
```

---

# نمودارهای تولیدشده

نمودارهای زیر تولید می‌شوند:

* Throughput vs Goroutines
* Total Time vs Goroutines
* Average Latency vs Goroutines

برای هر workload نمودار جداگانه ایجاد می‌شود.

---

# مشاهده Go Trace

برای مشاهده رفتار Scheduler و goroutineها از ابزار `runtime/trace` استفاده شده است.

فایل trace در مسیر زیر ذخیره می‌شود:

```text
trace.out
```

برای مشاهده آن:

```bash
go tool trace trace.out
```

پس از اجرای دستور فوق، یک رابط وب محلی باز می‌شود که اطلاعاتی مانند موارد زیر را نمایش می‌دهد:

* وضعیت goroutineها
* زمان‌بندی Scheduler
* blocking
* utilization پردازنده‌ها
* syscallها
* garbage collection

---

# هدف آزمایش‌ها

این پروژه برای بررسی موارد زیر طراحی شده است:

* تاثیر افزایش تعداد goroutineها بر throughput
* تاثیر افزایش concurrency بر latency
* مقایسه رفتار سیستم در مقادیر مختلف `GOMAXPROCS` 
* مشاهده نقطه‌ای که افزایش همزمانی دیگر باعث بهبود عملکرد نمی‌شود
* بررسی تفاوت workloadهای CPU-bound و IO-bound

---
