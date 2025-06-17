#!/bin/node

/// <reference path="./types.d.ts" />
/// <reference path="./hook.d.ts" />

define(() => {
  const GiB = 1024 * 1024 * 1024
  const express = $app.findCollectionByNameOrId("express")
  /**
   * @param {core.Record} item
   */
  return function cb(item) {
    $app.runInTransaction((tx) => {
      let num = item.getInt("num")
      // 售价: 2元/GiB, 所以 1 元只有 0.5G
      let g = num * 0.5 * GiB
      let user = tx.findRecordById("users", item.getString("user"))
      let b = user.getInt("remaining_bytes")
      user.set("remaining_bytes", b + g)
      tx.save(user)
      // 要记录下变更前后的差值
      let order = tx.findRecordById("orders", item.getString("order"))
      let ex = new Record(express)
      ex.set("user", item.getString("user"))
      ex.set("order", item.getString("order"))
      ex.set("items", [item.id])
      let b1 = bytes2str(b)
      let b2 = bytes2str(b + g)
      ex.set(
        "value",
        JSON.stringify({
          已发放可用流量到用户帐号中: `发放前: ${b1}, 现在: ${b2}`,
        })
      )
      ex.set("remark", `发放前: ${b}, 现在: ${b + g}`)
      tx.save(ex)
      let exs = order.getStringSlice("express")
      exs.push(ex.id)
      order.set("express", exs)
      tx.save(order)
    })
  }
})
