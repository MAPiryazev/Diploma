from __future__ import annotations

from pathlib import Path

import win32com.client as win32


ROOT = Path(__file__).resolve().parents[1]
PRESENTATION = ROOT / "docs" / "презентация — копия.pptx"


MsoShapeRectangle = 1
MsoShapeRoundedRectangle = 5
MsoTextOrientationHorizontal = 1
PpLayoutBlank = 12
MsoFalse = 0
MsoTrue = -1


def rgb(red: int, green: int, blue: int) -> int:
    """PowerPoint COM uses the same integer layout as VBA RGB()."""
    return red + (green << 8) + (blue << 16)


BLUE = rgb(28, 64, 120)
LIGHT_BLUE = rgb(232, 240, 252)
GREEN = rgb(232, 246, 239)
GRAY = rgb(245, 247, 250)
DARK = rgb(35, 35, 35)
MUTED = rgb(90, 90, 90)
WHITE = rgb(255, 255, 255)


def clear_slide(slide) -> None:
    for idx in range(slide.Shapes.Count, 0, -1):
        slide.Shapes.Item(idx).Delete()


def slide_text(slide) -> str:
    parts: list[str] = []
    for idx in range(1, slide.Shapes.Count + 1):
        shape = slide.Shapes.Item(idx)
        try:
            if shape.HasTextFrame and shape.TextFrame.HasText:
                parts.append(shape.TextFrame.TextRange.Text)
        except Exception:
            continue
    return "\n".join(parts)


def remove_generated_slides_if_present(pres) -> None:
    """Make the script safe to run repeatedly on an already edited deck."""
    if pres.Slides.Count < 19:
        return
    if "Что реализовано в прототипе" not in slide_text(pres.Slides.Item(9)):
        return

    for idx in (11, 10, 9):
        pres.Slides.Item(idx).Delete()


def add_text(
    slide,
    text: str,
    left: float,
    top: float,
    width: float,
    height: float,
    size: int,
    color: int = DARK,
    bold: bool = False,
):
    shape = slide.Shapes.AddTextbox(
        MsoTextOrientationHorizontal,
        left,
        top,
        width,
        height,
    )
    shape.TextFrame.TextRange.Text = text
    shape.TextFrame.WordWrap = MsoTrue
    shape.TextFrame.AutoSize = 0
    shape.TextFrame.MarginLeft = 6
    shape.TextFrame.MarginRight = 6
    shape.TextFrame.MarginTop = 4
    shape.TextFrame.MarginBottom = 4
    shape.TextFrame.TextRange.Font.Name = "Arial"
    shape.TextFrame.TextRange.Font.Size = size
    shape.TextFrame.TextRange.Font.Color.RGB = color
    shape.TextFrame.TextRange.Font.Bold = MsoTrue if bold else MsoFalse
    return shape


def add_title(slide, width: float, title: str) -> None:
    bar = slide.Shapes.AddShape(MsoShapeRectangle, 0, 0, width, 54)
    bar.Fill.ForeColor.RGB = BLUE
    bar.Line.Visible = MsoFalse
    add_text(slide, title, 28, 9, width - 56, 38, 24, WHITE, True)


def add_footer(slide, width: float, height: float, num: int) -> None:
    add_text(
        slide,
        "ВКР (б) студента Пирязева Михаила",
        28,
        height - 28,
        360,
        18,
        9,
        MUTED,
    )
    add_text(slide, str(num), width - 45, height - 30, 24, 18, 9, MUTED)


def refresh_footer(slide, width: float, height: float, num: int) -> None:
    footer_bg = slide.Shapes.AddShape(MsoShapeRectangle, 0, height - 34, width, 34)
    footer_bg.Fill.ForeColor.RGB = WHITE
    footer_bg.Line.Visible = MsoFalse
    add_footer(slide, width, height, num)


def refresh_all_footers(pres, width: float, height: float) -> None:
    for idx in range(1, pres.Slides.Count + 1):
        refresh_footer(pres.Slides.Item(idx), width, height, idx)


def setup_slide(slide, width: float, height: float, title: str, num: int) -> None:
    clear_slide(slide)
    bg = slide.Shapes.AddShape(MsoShapeRectangle, 0, 0, width, height)
    bg.Fill.ForeColor.RGB = WHITE
    bg.Line.Visible = MsoFalse
    add_title(slide, width, title)
    add_footer(slide, width, height, num)


def add_bullets(
    slide,
    items: list[str],
    left: float,
    top: float,
    width: float,
    height: float,
    size: int = 18,
):
    text = "\r".join(f"• {item}" for item in items)
    shape = add_text(slide, text, left, top, width, height, size)
    shape.TextFrame.TextRange.ParagraphFormat.SpaceAfter = 8
    return shape


def add_box(
    slide,
    title: str,
    items: list[str],
    left: float,
    top: float,
    width: float,
    height: float,
    fill: int,
):
    box = slide.Shapes.AddShape(MsoShapeRoundedRectangle, left, top, width, height)
    box.Fill.ForeColor.RGB = fill
    box.Line.ForeColor.RGB = BLUE
    box.Line.Weight = 1.2
    add_text(slide, title, left + 14, top + 10, width - 28, 24, 16, BLUE, True)
    add_bullets(slide, items, left + 14, top + 42, width - 28, height - 54, 13)


def rewrite_existing_slides(pres, width: float, height: float) -> None:
    s = pres.Slides.Item(2)
    setup_slide(s, width, height, "Актуальность темы", 2)
    add_bullets(
        s,
        [
            "Финансовая операция в реальной инфраструктуре превращается в цепочку событий и статусов.",
            "При асинхронной обработке возникают повторы, задержки, неупорядоченность и частичные сбои.",
            "Для контроля качества обработки нужны не только CRUD-операции, но и событийный контур, защита от дублей и наблюдаемость.",
            "В работе реализован учебный backend-прототип, показывающий эти инженерные механизмы на транзакциях.",
        ],
        60,
        90,
        width - 120,
        250,
        20,
    )

    s = pres.Slides.Item(3)
    setup_slide(s, width, height, "Цель и задачи работы", 3)
    add_text(
        s,
        "Цель: спроектировать и реализовать backend-систему приёма, хранения и асинхронной обработки финансовых транзакций с возможностью оперативного мониторинга.",
        60,
        86,
        width - 120,
        58,
        18,
        DARK,
        True,
    )
    add_bullets(
        s,
        [
            "Спроектировать модель данных и HTTP API для жизненного цикла транзакций.",
            "Реализовать событийный конвейер на базе Kafka и transactional outbox.",
            "Обеспечить идемпотентность создания транзакций и дедупликацию Kafka-событий.",
            "Добавить retry, DLQ/replay и метрики Prometheus/Grafana для демонстрации устойчивости.",
        ],
        60,
        165,
        width - 120,
        250,
        18,
    )

    s = pres.Slides.Item(4)
    setup_slide(s, width, height, "Ограничения задачи", 4)
    add_bullets(
        s,
        [
            "Фокус работы — backend-компонент и демонстрационный событийный контур, а не полноценная банковская платформа.",
            "Не реализуются промышленный RBAC, PCI DSS, TLS/SASL для всех инфраструктурных соединений и сложная ledger-модель балансов.",
            "Стенд рассчитан на локальный запуск через Docker Compose без кластеризации PostgreSQL и Kafka.",
            "Ценность прототипа — в связке типовых механизмов: API, PostgreSQL, Kafka, outbox, consumer, DLQ и наблюдаемость.",
        ],
        60,
        92,
        width - 120,
        280,
        19,
    )

    s = pres.Slides.Item(5)
    setup_slide(s, width, height, "Порядок выполнения работы", 5)
    add_bullets(
        s,
        [
            "Проанализирована предметная область потоковой обработки финансовых транзакций.",
            "Спроектированы модель данных, слои приложения и контур доменных событий.",
            "Реализованы HTTP API, PostgreSQL-репозитории, audit log и Bearer-token авторизация.",
            "Добавлены Kafka producer/consumer, transactional outbox, deduplication, retry, DLQ и replay.",
            "Настроены Prometheus, Grafana dashboards и сценарии нагрузки через Makefile.",
        ],
        60,
        92,
        width - 120,
        290,
        18,
    )

    s = pres.Slides.Item(6)
    setup_slide(s, width, height, "Стек технологий", 6)
    add_box(s, "Backend и данные", ["Go", "PostgreSQL", "SQL-миграции", "Clean Architecture"], 50, 86, 270, 150, LIGHT_BLUE)
    add_box(s, "Событийный контур", ["Apache Kafka", "transactional outbox", "consumer group", "DLQ и replay"], 345, 86, 270, 150, GREEN)
    add_box(s, "Наблюдаемость и стенд", ["Prometheus", "Grafana dashboards", "kafka-exporter", "Docker Compose, Makefile"], 50, 260, 565, 120, GRAY)

    s = pres.Slides.Item(7)
    setup_slide(s, width, height, "Структура проекта и слои", 7)
    add_box(s, "cmd/", ["server — HTTP API и outbox relay", "consumer — обработка Kafka-событий", "dlqreplay — повторная публикация из DLQ", "load — генератор нагрузки"], 44, 84, 285, 250, LIGHT_BLUE)
    add_box(s, "internal/", ["handlers — HTTP transport", "service — сценарии использования", "repository — PostgreSQL", "events — envelope, publisher, outbox, DLQ"], 354, 84, 285, 250, GREEN)
    add_text(s, "Зависимости направлены к бизнес-логике: транспорт, БД, Kafka и метрики подключаются как внешние адаптеры.", 62, 360, width - 124, 44, 17, DARK, True)

    s = pres.Slides.Item(9)
    setup_slide(s, width, height, "Гарантии доставки сообщений", 9)
    add_bullets(
        s,
        [
            "Событие создаётся только для уже зафиксированного бизнес-изменения в PostgreSQL.",
            "Transactional outbox устраняет разрыв «запись в БД прошла, а событие в Kafka не ушло».",
            "Outbox relay публикует готовые события в Kafka topic transactions.events.",
            "Consumer подтверждает offset вручную — только после валидации, обработки и сохранения event_id.",
            "Семантика текущего контура: at least once + дедупликация по event_id.",
        ],
        60,
        90,
        width - 120,
        300,
        18,
    )

    s = pres.Slides.Item(10)
    setup_slide(s, width, height, "Идемпотентность и дедупликация", 10)
    add_box(s, "HTTP API", ["Idempotency-Key используется для POST /items", "Повтор того же body возвращает сохранённый ответ", "Тот же ключ + другое body = 409 Conflict"], 50, 92, 270, 210, LIGHT_BLUE)
    add_box(s, "Kafka consumer", ["Дедупликация выполняется по event_id", "Таблица processed_events хранит уже обработанные события", "Повторная доставка не искажает проекцию"], 345, 92, 270, 210, GREEN)
    add_text(s, "Важно: Kafka напрямую не участвует в HTTP-идемпотентности. Это два разных механизма защиты от повторов на разных уровнях системы.", 60, 330, width - 120, 54, 17, DARK, True)

    s = pres.Slides.Item(11)
    setup_slide(s, width, height, "DLQ: отдельный топик для сбоев", 11)
    add_bullets(
        s,
        [
            "Некорректные сообщения и ошибки после retry изолируются в transactions.events.dlq.",
            "Проблемное событие не блокирует основной поток обработки.",
            "В DLQ сохраняются исходный payload, причина сбоя и контекст Kafka-сообщения.",
            "Рост DLQ, retry и ошибок виден в Grafana, поэтому проблему можно обнаружить во время демонстрации.",
        ],
        60,
        94,
        width - 120,
        270,
        19,
    )

    s = pres.Slides.Item(12)
    setup_slide(s, width, height, "Replay из DLQ", 12)
    add_bullets(
        s,
        [
            "Replay выполняет отдельная утилита cmd/dlqreplay, а не основной API и не consumer.",
            "Запуск предполагается вручную после разбора и устранения причины ошибки.",
            "Утилита читает transactions.events.dlq и публикует исходный payload обратно в transactions.events.",
            "Поддерживаются ограничение количества сообщений и dry-run для безопасной проверки сценария.",
        ],
        60,
        96,
        width - 120,
        270,
        19,
    )

    s = pres.Slides.Item(13)
    setup_slide(s, width, height, "Тестирование и демонстрация", 13)
    add_bullets(
        s,
        [
            "Модульные тесты покрывают ключевые части: JWT/auth, envelope событий, replay и идемпотентность.",
            "Makefile содержит сценарии нагрузки: load-demo, load-events, load-errors, load-stress.",
            "Во время нагрузки проверяются HTTP latency, Kafka lag, retries, DLQ, ошибки producer/consumer.",
            "Демонстрационный сценарий показывает путь события от POST /items до Kafka consumer и /analytics/stream.",
        ],
        60,
        94,
        width - 120,
        280,
        18,
    )

    s = pres.Slides.Item(15)
    setup_slide(s, width, height, "Заключение", 15)
    add_bullets(
        s,
        [
            "Спроектирован и реализован учебный backend-прототип обработки финансовых транзакций.",
            "Синхронный контур: HTTP API, PostgreSQL, audit log, Bearer-token auth и идемпотентность POST /items.",
            "Асинхронный контур: transactional outbox, Kafka, consumer, deduplication, retry, DLQ и replay.",
            "Наблюдаемость: Prometheus-метрики и Grafana dashboards для API и Kafka pipeline.",
            "Проект не является production-ready банковской системой, но демонстрирует применимые индустриальные подходы.",
        ],
        60,
        90,
        width - 120,
        300,
        18,
    )


def add_new_slides(pres, width: float, height: float) -> None:
    s = pres.Slides.Add(9, PpLayoutBlank)
    setup_slide(s, width, height, "Что реализовано в прототипе", 9)
    add_box(s, "Синхронный контур", ["POST/GET/PUT/DELETE /items", "PostgreSQL как источник истины", "Idempotency-Key для создания", "audit log операций"], 42, 86, 280, 235, LIGHT_BLUE)
    add_box(s, "Асинхронный контур", ["event_outbox в той же SQL-транзакции", "outbox relay → Kafka", "consumer + processed_events", "retry, DLQ и replay"], 342, 86, 280, 235, GREEN)
    add_text(s, "Главная идея: Kafka не заменяет PostgreSQL, а используется как событийная шина после фиксации бизнес-изменения.", 58, 345, width - 116, 48, 18, DARK, True)

    s = pres.Slides.Add(10, PpLayoutBlank)
    setup_slide(s, width, height, "Transactional outbox: зачем нужен", 10)
    add_text(s, "Проблема без outbox: бизнес-операция уже записана в БД, но публикация события в Kafka может не выполниться из-за сбоя сети или брокера.", 56, 88, width - 112, 48, 17, DARK, True)
    add_box(s, "1. HTTP API", ["получает запрос", "валидирует данные"], 42, 166, 132, 120, LIGHT_BLUE)
    add_text(s, "→", 185, 205, 30, 40, 24, BLUE, True)
    add_box(s, "2. PostgreSQL TX", ["transactions", "event_outbox"], 220, 166, 150, 120, GREEN)
    add_text(s, "→", 382, 205, 30, 40, 24, BLUE, True)
    add_box(s, "3. Outbox relay", ["читает pending events", "публикует в Kafka"], 416, 166, 150, 120, GRAY)
    add_text(s, "→", 578, 205, 30, 40, 24, BLUE, True)
    add_box(s, "4. Kafka/consumer", ["transactions.events", "обработка и dedup"], 608, 166, 150, 120, LIGHT_BLUE)
    add_text(s, "Итог: изменение данных и событие фиксируются атомарно в PostgreSQL, а публикация в Kafka становится восстанавливаемым фоновым процессом.", 58, 330, width - 116, 58, 17, DARK)

    s = pres.Slides.Add(11, PpLayoutBlank)
    setup_slide(s, width, height, "Наблюдаемость и демонстрационный сценарий", 11)
    add_box(s, "Что видно в Grafana", ["HTTP throughput и latency", "producer/consumer throughput", "consumer lag по topic/partition", "DLQ, retries, errors", "outbox и consumer latency"], 46, 84, 285, 260, LIGHT_BLUE)
    add_box(s, "Как показывать на защите", ["запустить docker compose", "создать нагрузку через make load-events", "показать /analytics/stream", "открыть Diploma API Quality", "открыть Diploma Kafka Pipeline"], 358, 84, 285, 260, GREEN)
    add_text(s, "Смысл демонстрации: система не только выполняет операции, но и показывает состояние событийного конвейера под нагрузкой.", 58, 360, width - 116, 42, 17, DARK, True)


def rewrite_tail_slides(pres, width: float, height: float) -> None:
    if pres.Slides.Count < 19:
        return

    s = pres.Slides.Item(17)
    setup_slide(s, width, height, "QR-код репозитория проекта", 17)
    add_text(
        s,
        "Ссылка с исходным кодом доступна по QR-коду. Отсканируйте камерой для доступа к хранилищу кода проекта.",
        72,
        118,
        width - 144,
        90,
        22,
        DARK,
        True,
    )
    add_text(
        s,
        "Во время защиты этот слайд можно использовать как подтверждение воспроизводимости: в репозитории находятся код, миграции, Docker Compose, Makefile, дашборды Grafana и материалы демонстрации.",
        72,
        230,
        width - 144,
        110,
        18,
        DARK,
    )

    s = pres.Slides.Item(19)
    setup_slide(s, width, height, "Источники", 19)
    add_bullets(
        s,
        [
            "Приемы объектно-ориентированного проектирования.",
            "Таненбаум Э. С., Бос Х. Современные операционные системы.",
            "Кормен Т. Х., Лейзерсон Ч. Э., Ривест Р. Л., Штайн К. Алгоритмы: построение и анализ.",
            "Документация PostgreSQL, Apache Kafka, Prometheus и Grafana использовалась при реализации прототипа.",
        ],
        60,
        92,
        width - 120,
        250,
        18,
    )


def main() -> None:
    if not PRESENTATION.exists():
        raise FileNotFoundError(PRESENTATION)

    app = win32.Dispatch("PowerPoint.Application")
    presentation = app.Presentations.Open(str(PRESENTATION), False, False, False)
    try:
        remove_generated_slides_if_present(presentation)
        width = presentation.PageSetup.SlideWidth
        height = presentation.PageSetup.SlideHeight
        rewrite_existing_slides(presentation, width, height)
        add_new_slides(presentation, width, height)
        rewrite_tail_slides(presentation, width, height)
        refresh_all_footers(presentation, width, height)
        presentation.Save()
        print(f"Updated: {PRESENTATION}")
        print(f"Slides: {presentation.Slides.Count}")
    finally:
        presentation.Close()
        app.Quit()


if __name__ == "__main__":
    main()
