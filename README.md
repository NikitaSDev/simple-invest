# Описание проекта
#### Основное назначение
Веб-сервис позволяет определить некоторые финансовые показатели торгуемой облигации* на основании данных, получаемых от Мосбиржи. Рассчитываются показатели текущей и простой доходности, а так же текущая и простая доходности с учётом уплаты НДФЛ 13% с купонов и разницы цены и номинала с учётом льготы долгосрочного владения.

Эндпоинт - `bondindicators`, параметры: `isin` - код облигации, обязательный.

Формат получаемых данных - JSON, состав полей:
- Isin - код ценной бумаги
- FaceValue - текущая номинальная стоимость
- AccruedInt - НКД
- Coupon - сумма текущего купона
- PercentPrice - цена в процентах
- Price - цена
- DaysToEvent - дней до события (погашение или оферта)
- MatDate - дата погашения
- OfferDate - дата оферты
- SimpleYield - простая доходоность
- NetSimpleYield - простая доходность с учётом НДФЛ
- CurrentYield - Текущая доходность
- NetCurrentYield - текущая доходность с учётом НДФЛ
- MaturityTax - налог при погашении/выкупе по оферте

#### Дополнительные возможности:
- shares - возвращает JSON со списком торгуемых акций, параметры: `update`, тип - строка, значение - `yes`, необязательный. Данные получаются из БД, для загрузки данных с Мосбиржи необходимо установить параметр `update`.
- dividends - возвращает JSON со списком выплаченных дивидендов, параметры: `isin`, тип - строка, обязательный.
- bonds - возвращает JSON со списком торгуемых облигаций, параметры: `update`, тип - строка, значение - `yes`, необязательный. Данные получаются из БД, для загрузки данных с Мосбиржи необходимо установить параметр `update`.
- coupons - возвращает JSON со списком купонов, параметры: `isin`, тип - строка, обязательный.
- amortizations  - возвращает JSON со списком амортизационных выплат, параметры: `isin`, тип - строка, обязательный.

##### Необходимые условия
Наличие запущеной СУБД PostgreSQL с созаднными таблицами, указанными в `doc/DB doc`.
Порт для запуска - `7540`
##### Примечания
\* - только для для облигаций с фиксированным купоном
