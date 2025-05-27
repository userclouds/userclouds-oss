{{ config(materialized='table') }}

select email, sum(total) as total_order_value
from {{ ref('all_users') }} join {{ ref('orders') }} on all_users.id = orders.user_id
group by email
