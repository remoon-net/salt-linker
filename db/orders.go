package db

type OrderStatus string

const (
	OrderStatusWaitPay OrderStatus = "待支付"
	OrderStatusPaid    OrderStatus = "已支付"
	OrderStatusCount   OrderStatus = "已记录"
	OrderStatusMaking  OrderStatus = "制作中"
	OrderStatusSended  OrderStatus = "已发货"
	OrderStatusClosed  OrderStatus = "已关闭"
)

type PaymentStatus string

const (
	PaymentStatusCreated PaymentStatus = "订单已创建"
	PaymentStatusPaid    PaymentStatus = "订单已支付" //支付成功
	PaymentStatusClosed  PaymentStatus = "订单已关闭" //订单已关闭不允许再次支付
	PaymentStatusPaying  PaymentStatus = "订单支付中" //订单进入支付流程了, 再次支付需要关闭上一次生成的微信支付订单
)
