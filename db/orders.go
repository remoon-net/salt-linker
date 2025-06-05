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
