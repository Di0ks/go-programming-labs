package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/Di0ks/go-programming-labs/utils"
)

// 1. Структура `Person` и метод для вывода информации

// Структура описывающая человека (имя и возраст)
type Person struct {
	name string
	age  int
}

// Выводит информацию о человеке
func (p Person) print() {
	var age_s string
	rem := p.age % 10
	if rem > 0 && rem < 5 {
		age_s = "года"
	} else {
		age_s = "лет"
	}

	fmt.Printf("%s %d %s\n", p.name, p.age, age_s)
}

// 2. Метод "дня рождения"
//
// Увеличивает возраст на 1
func (p *Person) birthday() {
	fmt.Printf("С днём рождения, %s!!!🥳🎂🎉\n", p.name)
	p.age++
}

// 1. и 2. Создание структуры и проверка методов
func person_struct() {
	fmt.Println("Задание 1.")

	name := utils.ReadLine("Имя человека: ")
	age := utils.ReadSingleInt("Возраст: ")
	person := Person{name, age}
	fmt.Println("Структура создана")
	person.print()

	fmt.Println()

	fmt.Println("Задание 2.")

	person.birthday()
	fmt.Println("Теперь возраст равен", person.age)

	fmt.Println()
}

// 3. Структура `Circle` и метод для площади

// Окружность с радиусом
type Circle struct {
	radius float64
}

// Вычисляет площадь окружности
func (c Circle) area() float64 {
	return math.Pi * c.radius * c.radius
}

// 3. Проверка структуры `Circle` и метода
func circle_struct() {
	fmt.Println("Задание 3.")

	radius := utils.ReadSingleFloat[float64]("Введите радиус окружности: ")
	circ := Circle{radius}
	fmt.Println("Площадь равна", circ.area())

	fmt.Println()
}

// 4. Интерфейс `Shape` и метод `Area`

// Интерфейс фигуры для расчёта площади
type Shape interface {
	Area() float64
	// Метод имени для различия фигур
	Name() string
}

func (c Circle) Area() float64 {
	return c.area()
}

func (c Circle) Name() string {
	return "окружность"
}

// Прямоугольник
type Rectangle struct {
	length, width int
}

// Вычисляет площадь прямоугольника
func (rect Rectangle) Area() float64 {
	return float64(rect.length * rect.width)
}

func (rect Rectangle) Name() string {
	return "прямоугольник"
}

// Функция для получения фигуры от пользователя
func make_shape() Shape {
	choice := utils.ReadSingleInt("Какую фигуру создать (0 = прямоугольни, 1 = окружность): ")
	switch choice {
	case 0:
		nums := utils.ReadInts("Введите 2 числа через пробел — ширину и высоту прямоугольника: ")
		if len(nums) != 2 {
			panic("Нужно ввести 2 числа")
		}
		width, height := nums[0], nums[1]
		return Rectangle{width, height}
	case 1:
		radius := utils.ReadSingleFloat[float64]("Введите радиус окружности: ")
		return Circle{radius}
	default:
		panic("Неверный выбор фигуры, нужно ввести 0 или 1")
	}
}

// 4. и 5. Площади объектов с интерфейсом
func shapes_areas(shapes []Shape) {
	for _, shape := range shapes {
		fmt.Printf(
			"Фигура '%s' иммет площадь %f\n",
			shape.Name(),
			shape.Area(),
		)
	}
}

// 6. Интерфейс `Stringer` и его реализация на `Book`

// Книга, содержащая информацию о названии, авторе, годе выпуска и страницы
type Book struct {
	author, name string
	release_year int
	pages        []string
}

// Интерфейс превращения в строку
type Stringer interface {
	// Превращает объект в строку
	Stringify() string
}

func (b Book) Stringify() string {
	var pages strings.Builder
	for i, page := range b.pages {
		_, err := fmt.Fprintf(
			&pages,
			"-----страница %d-----\n%s\n\n",
			i+1,
			page,
		)
		if err != nil {
			panic(fmt.Sprintf("Не удалось собрать страницы книги: %v", err))
		}
	}
	return fmt.Sprintf(
		"Книга «%s»\nАвтор: %s\nВыпущена в %d году\nТекст:\n%s",
		b.name,
		b.author,
		b.release_year,
		pages.String(),
	)
}

// 6. Создание книги и использование интерфейса `Stringere`
func stringer_usage() {
	fmt.Println("Задания 6.")

	fmt.Println("Время написать книгу!")
	name := utils.ReadLine("Как она будет называться?\n>> ")
	author := utils.ReadLine("Кто автор?\n>> ")
	release_year := utils.ReadSingleInt("В каком году выпущена?\n>> ")
	fmt.Println(`Теперь нужно написать текст книги.
Правила ввода:
- нажмите Enter два раза, чтобы добавить новую строку;
- напишите <#page#>, чтобы начать новую страницу;
- напишите <#stop#>, чтобы завершить набор текста.`)

	pages := make([]string, 0, 1)
page_loop:
	for page_index := 0; ; page_index++ {
		fmt.Printf("<страница %d>\n> ", page_index+1)
		var page strings.Builder
	line_loop:
		for {
			line := utils.ReadLine("")
			if len(line) == 0 {
				_, err := page.WriteRune('\n')
				if err != nil {
					panic(fmt.Sprintf("не удалось добавить новую строку: %v", err))
				}
				fmt.Println("<добавлена пустая строка>")
				continue line_loop
			}
			if line == "<#page#>" {
				pages = append(pages, page.String())
				continue page_loop
			}
			if line == "<#stop#>" {
				pages = append(pages, page.String())
				break page_loop
			}
			_, err := page.WriteString(line)
			if err != nil {
				panic(fmt.Sprintf("не удалось добавить новую строку: %v", err))
			}
			_, err = page.WriteRune(' ')
			if err != nil {
				panic(fmt.Sprintf("не удалось добавить пробел при новой строчке: %v", err))
			}
		}
	}

	book := Book{author, name, release_year, pages}
	fmt.Println("\nКнига успешно создана")
	fmt.Println(book.Stringify())

	fmt.Println()
}

func main() {
	person_struct()
	circle_struct()

	fmt.Println("Задания 4. и 5.")

	count := utils.ReadSingleInt("Сколько фигур сделать: ")
	shapes := make([]Shape, count)
	for i := range count {
		fmt.Printf("#%d: ", i)
		shapes[i] = make_shape()
	}
	shapes_areas(shapes)
	fmt.Println()

	stringer_usage()
}
