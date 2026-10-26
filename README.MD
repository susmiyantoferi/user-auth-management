## Set up local :

- Clone repo
- Rename file config.yml.example to config.yml
- Database use is Postgresql only
- Change the contents of the config.yml file according to the database name, username, password, host, port, redis address, redis password, username email, password email respectively.

  ```bash
  postgres:
  host: "127.0.0.1"
  port: "5432"
  username: "postgres"
  password: "Terserah123"
  dbname: "auth"
  idleconnect: 10
  maxconnect: 100
  lifeconnect: 300

  redis:
  host: "127.0.0.1"
  port: "6379"
  db: 0
  password:
  username:

  email:
  username: "example@email"
  password: "hsjbj"
  ```

- Run in terminal

  ```bash
  go run cmd/main.go
  ```

- URL Documentation API : https://documenter.getpostman.com/view/50367071/2sBYHUULSz

NOTE : 
- pada saat user melakukan nonactive akun semua credensial token akan invalid
- pada saat user melakukan request forgot password setiap 1 menit hanya dapat melakukan request 1 kali
- pada saat verify otp max hit 5 kali dan setiap 1 menit hanya dapat sekali hit
- jika waktu register tidak menerima email otp bisa melakukan resend email otp
- resend otp dalam 1 menit hanya bisa dilakukan 1 kali
- semua credensial di save di redis dengan time to live masing masing