board {
  workflows = [
    workflow.add_product,
    workflow.checkout_order,
  ]
}

bounded_context "store" {
  title = "Store"

  aggregate "product" {
    title = "Product"
  }

  field_type "id" {
    type         = "Int"
    id_attribute = true
    example      = 1
  }

  event "product_added" {
    title     = "Product Added"
    aggregate = aggregate.product
  }
}
