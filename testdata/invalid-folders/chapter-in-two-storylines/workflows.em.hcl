chapter "ordering" {
  title     = "Ordering"
  workflows = [workflow.place_order]
}

storyline "sales" {
  title    = "Sales"
  chapters = [chapter.ordering]
}

state_change "place_order" {
  title = "Place Order"

  command "place_order_cmd" {
    title     = "Place Order"
    aggregate = aggregate.order.order
    to        = [event.order.order_placed]
  }
}
