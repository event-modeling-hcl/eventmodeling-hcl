bounded_context "orders" {
  title = "Customer purchases"

  field_type "order_id" {
    type = "UUID"
  }

  event "order_placed" {
    field "order_id" {}
  }
}

system "partner" {
  external = true
}

bounded_context "partner" {
  owner    = system.partner
  external = true

  field_type "partner_order_id" {
    type = "UUID"
  }

  event "partner_order_shared" {
    field "partner_order_id" {}
  }
}

actor "customer" {
  auth_required = true
}

state_change "place_order" {
  title = "Checkout purchase"

  screen "checkout" {
    actor = actor.customer
    to    = [command.place_order]
    field "order_id" {}
  }

  command "place_order" {
    to = [event.orders.order_placed]
    field "order_id" {}
  }

  scenario "order_is_placed" {
    when { command = command.place_order }
    then { event = event.orders.order_placed }
  }
}

state_view "view_order" {
  readmodel "order_summary" {
    question = "Which order was placed?"
    from     = [event.orders.order_placed]
    to       = [screen.order_summary]
    field "order_id" {}
  }

  screen "order_summary" {
    actor = actor.customer
  }

  scenario "placed_order_is_visible" {
    given { event = event.orders.order_placed }
    then { readmodel = readmodel.order_summary }
  }
}

translation "import_partner_order" {
  description = "The partner order identifier is the receiving domain's order identifier; this is a rename, not a newly generated value."

  processor "order_importer" {
    from = [event.partner.partner_order_shared]
    to   = [command.import_order]
    field "partner_order_id" {}
  }

  command "import_order" {
    description = "Rename partner_order_id from the incoming partner fact to order_id."
    to          = [event.orders.order_placed]
    field "order_id" {}
  }

  scenario "partner_order_is_imported" {
    given { event = event.partner.partner_order_shared }
    when { processor = processor.order_importer }
    then { event = event.orders.order_placed }
  }
}
