https://www.jianshu.com/p/40f9213b702f

https://www.bookstack.cn/read/echo-v3-zh/guide-response.md

http://www.voidcn.com/article/p-hezofvhc-bqy.html

https://github.com/rs/cors
https://github.com/thoas/stats


% curl -X POST  http://localhost:1323/users -d '{"ID":"2","name":"names"}'
{"id":"","name":""}


 % curl http://localhost:1323/users/1
{"id":"1","name":"Wreck-It Ralph"}

 % curl http://localhost:1323/users
{"1":{"id":"1","name":"Wreck-It Ralph"}}