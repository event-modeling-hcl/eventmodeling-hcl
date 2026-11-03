board {
  workflows = [workflow.place_order]
}

actor "customer" {
  auth_required = true
}

bounded_context "orders" {
  field_type "order_id" {
    type = "UUID"
  }

  event "order_placed" {
    fields = [field_type.order_id]
  }
}

state_change "place_order" {
  description = "The customer supplies order_id through checkout. Keep this workflow's behavior unchanged."

  screen "checkout" {
    actor  = actor.customer
    to     = [command.place_order]
    fields = [field_type.orders.order_id]
  }

  command "place_order" {
    to = ["event.orders.order_placed"]
    field "order_id" {}
  }

  scenario "order_is_placed" {
    when { command = command.place_order }
    then { event = event.orders.order_placed }
  }
}
