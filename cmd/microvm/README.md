# MicroVM

```console
$ echo 'wttr.in' \
  | sudo yaman c run --rm --interactive docker.io/library/alpine -- xargs wget -qO /dev/stdout \
  | sudo yaman c run --interactive --runtime microvm quay.io/aptible/alpine -- head -n 7
Weather report: Brussels, Belgium

     \  /       Partly cloudy
   _ /"".-.     17 °C
     \_(   ).   ↘ 24 km/h
     /(___(__)  10 km
                0.0 mm

$ sudo yaman c ls -a
CONTAINER ID                       IMAGE                           COMMAND     CREATED          STATUS                      PORTS
26127b728da94fd7a184549f2c0f586c   quay.io/aptible/alpine:latest   head -n 7   15 seconds ago   Exited (0) 12 seconds ago
$ sudo yaman c logs 26127b728da94fd7a184549f2c0f586c
Weather report: Brussels, Belgium

     \  /       Partly cloudy
   _ /"".-.     +22(24) °C
     \_(   ).   ↓ 7 km/h
     /(___(__)  10 km
                0.0 mm
```
