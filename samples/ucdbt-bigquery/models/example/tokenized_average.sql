select email, avg(amount) as average_order_amount
from {{ ref('tokenized_users') }} join {{ ref('stg_orders') }} on tokenized_users.id = stg_orders.user_id
group by email

