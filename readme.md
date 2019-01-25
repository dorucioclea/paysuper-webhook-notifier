Documentation will be soon

0. Когда мы хотим отправлять мы шлем в test и там, если всё хорошо
то событие обработают, а если плохо (т.е. basic.reject, basic.nack false). Никаких
приколов с временем в бащовой очереди нет.

1. exchange test (topic) - publisher
2. queue test.queue - subscriber

"x-dead-letter-exchange": "test.timeout10",

3. exchange test.timeout10 (topic)
4. queue test.queue.timeout10

"x-dead-letter-exchange":    "test",
"x-dead-letter-routing-key": "*",

теперь если на шаге 2 я не смог обоработать я должен поставить сообщению
заголовок ттл и сделать ему реджект.

5. мне надо бинд 3 и 4 