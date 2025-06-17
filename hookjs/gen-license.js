#!/bin/node

/// <reference path="./types.d.ts" />

/**
 * @typedef {Object} OrderItem
 * @prop {string} id
 * @prop {string} user
 * @prop {string} order
 * @prop {string} goods
 * @prop {number} num
 */

/**
 * @typedef {Object} Order
 * @prop {string} id
 * @prop {{[k:string]:{pubkey:string}}} address
 */

define(() => {
  const express = $app.findCollectionByNameOrId("express")
  /**
   * @param {core.Record} item
   */
  return function cb(item) {
    $app.runInTransaction((tx) => {
      let order = tx.findRecordById("orders", item.getString("order"))
      let addrs = JSON.parse(order.getString("address"))
      let addr = addrs[item.id]
      let l = GenLicense(addr.pubkey)
      let ex = new Record(express)
      ex.set("user", item.getString("user"))
      ex.set("order", item.getString("order"))
      ex.set("items", [item.id])
      ex.set("value", JSON.stringify({ license: l }))
      ex.set("remark", "salt-weblink的许可签名")
      tx.save(ex)
      let exs = order.getStringSlice("express")
      exs.push(ex.id)
      order.set("ex", exs)
      tx.save(order)
    })
  }
})
