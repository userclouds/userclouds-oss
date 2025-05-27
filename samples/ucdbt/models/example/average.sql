{{ config(materialized='table') }}

select email, avg(total) as average_order_total
from {{ ref('all_users') }} join {{ ref('orders') }} on all_users.id = orders.user_id
group by email
