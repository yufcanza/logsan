# logsan

Тесты и бенчмарки выполнялись на следующей конфигурации устройства:

 Характеристика | Значение |
|---|---|
| **ОС** | Windows 10 Pro 22H2 (19045) |
| **CPU** |  AMD Ryzen 5 2500U with Radeon Vega Mobile Gfx |
| **Оперативная память** | 16 GB DDR4 |
| **Диск** | Samsung SSD 970 EVO Plus 500GB (NVMe) |
| **Go версия** | go1.26.4 windows/amd64 |



# Назначение проекта:

Проект обезличивает логи для любых целей. Поддерживается гибкая настройка через конфигурацию детекторов. Генерирует отчёты в форматах JSON и Markdown.

# Сборка и запуск:

Клонирование репозатория:
git clone https://github.com/yufcanza/logsan.git
cd logsan

Сборка:
make build

Запуск:

Основная команда:
./logsan sanitize --in ./logs --out ./clean-logs --config detectors.yaml --report report.json

Пробный запуск (без записи):
./logsan dry-run --in ./logs --config detectors.yaml --report report.md

С сохранением словаря замен:
./logsan sanitize --in ./logs --out ./clean-logs -mapping-in mapping.json -mapping-in mapping_new.json

# Формат входных данных:

Поддерживает файлы с расширением .log и .txt
Пример строки:
2026-07-13 10:15:22 INFO user=ivanov email=ivanov@example.com ip=10.1.2.3 token=ab12cd34ef56 path=C:\Users\Ivanov\Documents\file.log
# Формат выходных данных:

Каждая строка обезличена
Пример строки:
2026-07-13 10:15:22 INFO user=ivanov email=email_001 ip=ip_001 token=token_001 path=C:\Users\user_001\Documents\file.log

# Примеры команд:
Базовый запуск:

logsan sanitize --in ./logs --out ./clean-logs

С формированием отчета:

./logsan sanitize --in ./logs --out ./clean-logs --report report.json

Со своими детекторами:

./logsan sanitize --in ./logs --out ./clean-logs --config my_detectors.yaml --report report.json

Пробный запуск: 

./logsan dry-run --in ./logs --config detectors.yaml --report dry-report.json

Запуск через Makefile: поддерживает следующие команды:

make build   - Сборка
make test    - Тесты
make bench   - Тестовый бенчмарк
make bench-1gb - Бенчмарк на 1 ГБ синтетического лога
make demo    - Демонстрация
make check-demo - Проверка работы демонстрации

# Краткое описание алгоритма:

1. Загрузка конфигурации - читается YAML-файл с детекторами, компилируются регулярные выражения.

2. Обход файлов - программа проходит по всем файлам во входной директории.

3. Параллельная обработка - каждый файл обрабатывается в отдельной горутине.

4. Построчная обработка - для каждой строки применяются все включённые детекторы.

5. Замена на псевдонимы - каждое найденное значение заменяется на стабильный псевдоним (email_001, ip_001 и т.д.).

6. Сбор статистики - подсчитывается количество замен по каждому детектору.

7. Генерация отчёта - сохраняется статистика и примеры замен (без раскрытия исходных значений).

# Известные ограничения реализации:
 Ограничение | Описание |
|---|---|
| Производительность | Скорость обработки ~6 МБ/с, что ниже требуемых |

# Результаты контрольного запуска:


  Файл: 1073741836 байт (1024.00 MB)
  Строк: 9215856
   Время: 186.95 секунд
    Скорость: 5.48 MB/s
