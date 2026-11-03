board {
  workflows = [workflow.place_order, workflow.view_order]
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
  status = "created"

  screen "checkout" {
    actor  = actor.customer
    to     = [command.place_order]
    fields = [field_type.orders.order_id]
  }

  command "place_order" {
    to     = [event.orders.order_placed]
    fields = [field_type.orders.order_id]
  }

  scenario "order_is_placed" {
    when { command = command.place_order }
    then { event = event.orders.order_placed }
  }
}

state_view "view_order" {
  status = "review"

  readmodel "order_summary" {
    question = "Which order was placed?"
    from     = [event.orders.order_placed]
    to       = [screen.order_summary]
    fields   = [field_type.orders.order_id]
  }

  screen "order_summary" {
    actor = actor.customer
  }

  scenario "placed_order_is_visible" {
    given {
      event = event.orders.order_placed
      field "order_id" {
        type    = field_type.orders.order_id
        example = "0e42f9d1-2f55-4cc6-b9e2-3d1926bd21bf"
      }
    }
    then {
      readmodel = readmodel.order_summary
      field "order_id" {
        type    = field_type.orders.order_id
        example = "0e42f9d1-2f55-4cc6-b9e2-3d1926bd21bf"
      }
    }
  }
}
