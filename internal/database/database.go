package database

import (
	"database/sql"
	"log"
	"slices"
	"sort"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/lib/pq"
)

func ConnectBase() *sql.DB {
	connStr := "user=pixart dbname=anonchat password=Aq28am#ud sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	pingErr := db.Ping() // Проверка соеденения с БД
	if pingErr != nil {
		log.Fatal(pingErr)
	}

	log.Println("База данных успешно подключена.")
	return db
}

// Создаёт пользователя в БД
func CreateUser(userID int64, db *sql.DB) {
	_, err := db.Exec("INSERT INTO users(user_ID) VALUES($1) ON CONFLICT (user_id) DO UPDATE SET user_id=$1;", userID)
	if err != nil {
		go log.Println(err)
	}
	for i := 1; i <= 5; i++ {
		_, err = db.Exec("INSERT INTO users_categories(user_id, category_id) VALUES($1, $2) ON CONFLICT (user_id, category_id) DO UPDATE SET category_id=$2;", userID, i)
		if err != nil {
			go log.Println(err)
		}
	}
	for i := 1; i <= 2; i++ {
		_, err = db.Exec("INSERT INTO users_genders(user_ID, gender_id) VALUES($1, $2) ON CONFLICT (user_id, gender_id) DO UPDATE SET gender_id=$2;", userID, i)
		if err != nil {
			go log.Println(err)
		}
	}
}

// Задает возрастную категорию пользователя в БД
func SetAge(userID int64, age int, db *sql.DB) {
	var category int
	switch { // Перевод возраста в категорию
	case age <= 17:
		category = 1
	case age >= 18 && age <= 25:
		category = 2
	case age >= 26 && age <= 33:
		category = 3
	case age >= 34 && age <= 41:
		category = 4
	case age >= 42:
		category = 5
	}
	_, err := db.Exec("UPDATE users SET age_category=$1 WHERE user_id=$2;", category, userID)
	if err != nil {
		go log.Println(err)
	}
}

// Задает пол пользователя в БД
func SetSex(userID int64, sex string, db *sql.DB) {
	_, err := db.Exec("UPDATE users SET gender=$1 WHERE user_id=$2;", sex, userID)
	if err != nil {
		go log.Println(err)
	}
}

// Получает пол и возраст пользователя из БД (age, sex, err)
func GetSexAge(bot *tgbotapi.BotAPI, userID int64, db *sql.DB) (int, string, error) {
	var sex string
	var age_category int

	rows, err := db.Query("SELECT COALESCE(age_category, 0), COALESCE(gender, 'unknown') FROM users WHERE user_id=$1", userID)
	if err != nil {
		return 0, "", err
	}
	defer func(rows *sql.Rows) error {
		err := rows.Close()
		if err != nil {
			return err
		}
		return nil
	}(rows)

	for rows.Next() {
		err := rows.Scan(&age_category, &sex)
		if err != nil {
			return 0, "unknown", err
		}
	}

	err = rows.Err()
	if err != nil {
		return 0, "unknown", err
	}

	return age_category, sex, nil
}

// Задает возрастной фильтр для пользователя в БД
func SetAgeFilter(bot *tgbotapi.BotAPI, userID int64, category string, db *sql.DB) {
	_, err := db.Exec("INSERT INTO users_categories(user_id, category_id) VALUES($1, $2) ON CONFLICT (user_id, category_id) DO UPDATE SET category_id=$2", userID, category)
	if err != nil {
		go log.Println(err)
	}
}

// Удаляет возрастной фильтр для пользователя в БД
func DeleteAgeFilter(bot *tgbotapi.BotAPI, userID int64, category int, db *sql.DB) {
	_, err := db.Exec("DELETE FROM users_categories WHERE category_id=$1 and user_id=$2;", category, userID)
	if err != nil {
		go log.Println(err)
	}
}

// Задает половой фильтр для пользователя в БД
func SetSexFilter(bot *tgbotapi.BotAPI, userID int64, gender string, db *sql.DB) {
	var resGender int

	switch gender {
	case "мужчина":
		_, err := db.Exec("DELETE FROM users_genders WHERE user_id=$1 AND gender_id=2", userID)
		if err != nil {
			go log.Println(err)
		}
		resGender = 1
	case "женщина":
		_, err := db.Exec("DELETE FROM users_genders WHERE user_id=$1 AND gender_id=1", userID)
		if err != nil {
			go log.Println(err)
		}
		resGender = 2
	case "любой":
		_, err := db.Exec("INSERT INTO users_genders(user_id, gender_id) VALUES($1, 1) ON CONFLICT (user_id, gender_id) DO UPDATE SET gender_id=1;", userID)
		if err != nil {
			go log.Println(err)
		}
		_, err = db.Exec("INSERT INTO users_genders(user_id, gender_id) VALUES($1, 2) ON CONFLICT (user_id, gender_id) DO UPDATE SET gender_id=2;", userID)
		if err != nil {
			go log.Println(err)
		}
		return
	}
	
	_, err := db.Exec("INSERT INTO users_genders(user_id, gender_id) VALUES($1, $2) ON CONFLICT (user_id, gender_id) DO UPDATE SET gender_id=$2;", userID, resGender)
	if err != nil {
		go log.Println(err)
	}
}

// Получает половой и возрастной фильтр пользователя из БД (sex, age, err)
func GetSexAgeFilter(bot *tgbotapi.BotAPI, userID int64, db *sql.DB) ([]string, []int, error) {
	sexArr := make([]string, 0, 2)
	ageCategoryArr := make([]int, 0, 5)

	rows, err := db.Query("SELECT COALESCE(search_categories.category, 0), COALESCE(search_genders.gender, '') FROM search_categories FULL JOIN users_categories ON search_categories.category_id=users_categories.category_id FULL JOIN users ON users.user_id=users_categories.user_id FULL JOIN users_genders ON users_genders.user_id=users.user_id FULL JOIN search_genders ON search_genders.gender_id=users_genders.gender_id WHERE users.user_id=$1;", userID)
	if err != nil {
		return []string{}, []int{}, err
	}
	defer func(rows *sql.Rows) error {
		err := rows.Close()
		if err != nil {
			return err
		}
		return nil
	}(rows)

	for rows.Next() {
		var sex string
		var age int

		err := rows.Scan(&age, &sex)
		if err != nil {
			return []string{}, []int{}, err
		}
		if !slices.Contains(sexArr, sex) {
			sexArr = append(sexArr, sex)
		}
		if !slices.Contains(ageCategoryArr, age) {
			ageCategoryArr = append(ageCategoryArr, age)
		}
		sort.SliceStable(ageCategoryArr, func(i, j int) bool {
			return ageCategoryArr[i] < ageCategoryArr[j]
		})
	}
	if rows.Err() != nil {
		return []string{}, []int{}, err
	}

	return sexArr, ageCategoryArr, nil
}

// Ищет собеседника (Обновляет is_search на true и ищет подходящих собеседников с такой-же колонкой)
func SearchCompanion(bot *tgbotapi.BotAPI, userID int64, sex []string, age []int, db *sql.DB) [2]int64 {
	var rows *sql.Rows

	_, err := db.Exec("UPDATE users SET is_search=$1, time_search=$2 WHERE user_id=$3;", true, time.Now().Unix(), userID)
	if err != nil {
		go log.Println(err)
	}

	myAge, MySex, err := GetSexAge(bot, userID, db)
	if err != nil {
		go log.Println(err)
	}

	if myAge != 0 || MySex != "unknown" { // Если пользователь установил свой пол и возраст
		if len(sex) == 2 && (len(age) == 5 || (len(age) == 1 && age[0] == 0)) { // Если пользователь ищет оба пола и все категории (или ни одну категорию)
			rows, err = db.Query(`SELECT users.user_id FROM users 
			LEFT JOIN users_categories ON users_categories.user_id=users.user_id
			LEFT JOIN search_categories ON search_categories.category_id=users_categories.category_id 
			JOIN users_genders ON users_genders.user_id=users.user_id
			JOIN search_genders ON search_genders.gender_id=users_genders.gender_id
			WHERE (search_categories.category IS NULL OR search_categories.category=$1) 
			AND search_genders.gender=$2 AND users.is_search=true ORDER BY users.time_search LIMIT 1;`, myAge, MySex)
			if err != nil {
				go log.Println(err)
			}
		} else { // Если пользователь ищет по особенным фильтрам
			rows, err = db.Query(`SELECT users.user_id FROM users 
			JOIN users_categories ON users_categories.user_id=users.user_id
			JOIN search_categories ON search_categories.category_id=users_categories.category_id 
			JOIN users_genders ON users_genders.user_id=users.user_id
			JOIN search_genders ON search_genders.gender_id=users_genders.gender_id 
			WHERE users.age_category=ANY($1) AND users.gender=ANY($2) AND search_categories.category=$3
			AND search_genders.gender=$4 AND users.is_search=true ORDER BY users.time_search LIMIT 1;`, pq.Array(age), pq.Array(sex), myAge, MySex)
			if err != nil {
				go log.Println(err)
			}
		}
	} else { // Если пользователь не установил свой пол и возраст
		rows, err = db.Query(`SELECT users.user_id FROM users 
		JOIN users_categories ON users_categories.user_id=users.user_id
		JOIN search_categories ON search_categories.category_id=users_categories.category_id 
		JOIN users_genders ON users_genders.user_id=users.user_id
		JOIN search_genders ON search_genders.gender_id=users_genders.gender_id 
		WHERE search_categories.category IN (1, 2, 3, 4, 5) AND search_genders.gender IN ('m', 'f') AND users.is_search=true 
		GROUP BY users.user_id
		HAVING COUNT(users.user_id)=10 ORDER BY users.time_search LIMIT 1;`)
		if err != nil {
			go log.Println(err)
		}
	}
	defer func(rows *sql.Rows) error {
		err := rows.Close()
		if err != nil {
			return err
		}
		return nil
	}(rows)

	for rows.Next() {
		var user_id int64
		rows.Scan(&user_id)
		if user_id != userID {
			StopSearch(bot, userID, db)
			StopSearch(bot, user_id, db)
			return [2]int64{user_id, userID}
		}
	}

	return [2]int64{}
}

// Остановить поиск (Обновляет is_search на false)
func StopSearch(bot *tgbotapi.BotAPI, userID int64, db *sql.DB) {
	_, err := db.Exec("UPDATE users SET is_search=$1 WHERE user_id=$2;", false, userID)
	if err != nil {
		go log.Println(err)
	}
}

// Возвращает true, если пользователь ищет собеседника на данный момент
func IsSearch(bot *tgbotapi.BotAPI, userID int64, db *sql.DB) bool {
	var is_search bool

	row := db.QueryRow("SELECT is_search FROM users WHERE user_id=$1", userID)
	row.Scan(&is_search)

	return is_search
}
