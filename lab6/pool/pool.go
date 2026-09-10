// Пакет реализует равномерное распределение работы на пул воркеров
package pool

import (
	"log"
)

type jobId int

// Пользовательская функция для выполнения
type JobFunc[Arg, Out any] func(Arg) Out

// сообщение об успешном выполнении пользовательской функции
type complete[T any] struct {
	id     jobId
	result T
}

// сам пул воркеров
type pool[Arg, Out any] struct {
	// хранит айдишники ещё не завершённых работ
	jobs map[jobId]struct{}
	// пользовательская функция
	fn JobFunc[Arg, Out]
	// канал для отправки воркерами результатов
	results chan complete[Out]
	// исходный данные, переданные пользователем
	sources []Arg
	// включить очень подробное логгирование
	trace bool
}

// горячий цикл выполнения пользовательской функции
func do_job[Arg, Out any](
	id jobId,
	// отсюда приходят данные для обработки
	in chan Arg,
	// сюда отправляются результаты выполнения
	out chan complete[Out],
	// для отправки сигнала о завершении работы воркера
	exit chan jobId,
	// пользовательская функция
	fn JobFunc[Arg, Out],
) {
	for val := range in {
		to_send := complete[Out]{
			id:     id,
			result: fn(val),
		}
		out <- to_send
	}

	exit <- id
}

// логгировать если включено подробное
func (p *pool[Arg, Out]) t(format string, v ...any) {
	if p.trace {
		log.Printf(format, v...)
	}
}

// ядро пула: здесь создаются основные каналы внутренней
// коммуникации и распределяется работа
func (p *pool[Arg, Out]) run(out chan Out) {
	data_stream := make(chan Arg)
	exited_jobs := make(chan jobId)

	log.Printf("Создаем %d воркеров...", len(p.jobs))
	for id := range p.jobs {
		go do_job(
			id,
			data_stream,
			p.results,
			exited_jobs,
			p.fn,
		)
		p.t(
			"[TRACE]: создан воркер #%d\n",
			id,
		)
	}

	// запускаем горутину для обработки результатов
	go func() {
		for res := range p.results {
			p.t(
				"[TRACE]: результат от #%d: %#v\n",
				res.id,
				res.result,
			)
			out <- res.result
		}
		close(out)
	}()

	// раздаем работу
	for _, item := range p.sources {
		p.t(
			"[TRACE]: отправлено: %#v\n",
			item,
		)
		data_stream <- item
	}
	close(data_stream)

	// проверяем на воркеров, завершивших свою работу, чтобы можно
	// было закрыть канал с результами и грациозно завершить работу
	// всего пула
	for len(p.jobs) > 0 {
		job := <-exited_jobs
		delete(p.jobs, job)
		p.t(
			"[TRACE]: сигнал о завершении от воркера #%d\n",
			job,
		)
	}
	close(p.results)
	log.Printf("Завершение работы пула...")
}

// Создает пул и запускает его, создавая указанное количество воркеров.
//
// Функция `fn` получает данные по одному из `data` в случайном порядке.
//
// Возвращает канал, отправлящий результаты по мере поступления.
func Spawn[Arg, Out any](
	// Количество воркеров
	worker_count int,
	// Данные для обработки
	data []Arg,
	// Функция для обработки
	fn JobFunc[Arg, Out],
	// Включить очень подробное логгирование
	trace bool,
) chan Out {
	ch := make(chan complete[Out])
	jobs := make(map[jobId]struct{}, worker_count)
	for i := range worker_count {
		jobs[jobId(i)] = struct{}{}
	}
	pool := new(pool[Arg, Out]{
		jobs:    jobs,
		sources: data,
		results: ch,
		fn:      fn,
		trace:   trace,
	})

	res := make(chan Out)
	go pool.run(res)
	return res
}
