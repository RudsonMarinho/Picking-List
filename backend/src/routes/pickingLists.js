const express = require("express");
const prisma = require("../lib/prisma");
const { authRequired } = require("../middleware/auth");

const router = express.Router();

router.use(authRequired);

router.get("/", async (req, res) => {
  const pickingLists = await prisma.pickingList.findMany({
    include: {
      order: true,
      items: { include: { product: true, location: true } },
    },
    orderBy: { createdAt: "desc" },
  });
  res.json(pickingLists);
});

router.get("/:id", async (req, res) => {
  const pickingList = await prisma.pickingList.findUnique({
    where: { id: req.params.id },
    include: {
      order: true,
      items: { include: { product: true, location: true } },
    },
  });

  if (!pickingList) {
    return res.status(404).json({ error: "Picking list não encontrada" });
  }

  res.json(pickingList);
});

router.post("/generate/:orderId", async (req, res) => {
  const { orderId } = req.params;

  try {
    const pickingList = await prisma.$transaction(async (tx) => {
      const order = await tx.order.findUnique({
        where: { id: orderId },
        include: { items: true, pickingList: true },
      });

      if (!order) {
        throw Object.assign(new Error("Pedido não encontrado"), { status: 404 });
      }
      if (order.pickingList) {
        throw Object.assign(new Error("Este pedido já possui uma picking list"), { status: 409 });
      }

      const itemsToCreate = [];
      const availableStockByProduct = new Map();

      for (const orderItem of order.items) {
        let remaining = orderItem.quantity;

        if (!availableStockByProduct.has(orderItem.productId)) {
          const stocks = await tx.stock.findMany({
            where: { productId: orderItem.productId, quantity: { gt: 0 } },
            orderBy: { quantity: "desc" },
          });
          availableStockByProduct.set(
            orderItem.productId,
            stocks.map((s) => ({ locationId: s.locationId, quantity: s.quantity }))
          );
        }

        const stocks = availableStockByProduct.get(orderItem.productId);

        for (const stock of stocks) {
          if (remaining <= 0) break;
          const take = Math.min(remaining, stock.quantity);
          if (take <= 0) continue;

          itemsToCreate.push({
            productId: orderItem.productId,
            locationId: stock.locationId,
            quantityRequested: take,
          });
          remaining -= take;
          stock.quantity -= take;
        }

        if (remaining > 0) {
          throw Object.assign(
            new Error(`Estoque insuficiente para o produto ${orderItem.productId}`),
            { status: 409 }
          );
        }
      }

      const created = await tx.pickingList.create({
        data: {
          orderId,
          items: { create: itemsToCreate },
        },
        include: { items: { include: { product: true, location: true } }, order: true },
      });

      await tx.order.update({ where: { id: orderId }, data: { status: "IN_PICKING" } });

      return created;
    });

    res.status(201).json(pickingList);
  } catch (err) {
    if (err.status) {
      return res.status(err.status).json({ error: err.message });
    }
    throw err;
  }
});

router.post("/:id/items/:itemId/pick", async (req, res) => {
  const { id, itemId } = req.params;
  const { quantityPicked } = req.body;

  if (typeof quantityPicked !== "number" || quantityPicked <= 0) {
    return res.status(400).json({ error: "quantityPicked (number > 0) é obrigatório" });
  }

  try {
    const result = await prisma.$transaction(async (tx) => {
      const item = await tx.pickingItem.findUnique({ where: { id: itemId } });
      if (!item || item.pickingListId !== id) {
        throw Object.assign(new Error("Item da picking list não encontrado"), { status: 404 });
      }

      const remainingToPick = item.quantityRequested - item.quantityPicked;
      if (quantityPicked > remainingToPick) {
        throw Object.assign(
          new Error(`Quantidade excede o restante a separar (${remainingToPick})`),
          { status: 409 }
        );
      }

      // Updates condicionais (updateMany + where) em vez de ler-depois-escrever: a
      // condição é avaliada atomicamente pelo banco no momento da escrita, evitando
      // que duas requisições concorrentes ultrapassem juntas o limite disponível.
      const stockUpdate = await tx.stock.updateMany({
        where: {
          productId: item.productId,
          locationId: item.locationId,
          quantity: { gte: quantityPicked },
        },
        data: { quantity: { decrement: quantityPicked } },
      });
      if (stockUpdate.count === 0) {
        throw Object.assign(new Error("Estoque insuficiente na localização"), { status: 409 });
      }

      const itemUpdate = await tx.pickingItem.updateMany({
        where: {
          id: itemId,
          pickingListId: id,
          quantityPicked: { lte: item.quantityRequested - quantityPicked },
        },
        data: { quantityPicked: { increment: quantityPicked } },
      });
      if (itemUpdate.count === 0) {
        throw Object.assign(
          new Error(`Quantidade excede o restante a separar (${remainingToPick})`),
          { status: 409 }
        );
      }

      const refreshedItem = await tx.pickingItem.findUnique({ where: { id: itemId } });
      const updatedItem = await tx.pickingItem.update({
        where: { id: itemId },
        data: {
          status: refreshedItem.quantityPicked >= refreshedItem.quantityRequested ? "PICKED" : "PENDING",
        },
        include: { product: true, location: true },
      });

      const allItems = await tx.pickingItem.findMany({ where: { pickingListId: id } });
      const allPicked = allItems.every((i) => i.status === "PICKED");

      let pickingList = await tx.pickingList.findUnique({ where: { id } });
      if (allPicked) {
        pickingList = await tx.pickingList.update({
          where: { id },
          data: { status: "COMPLETED", completedAt: new Date() },
        });
        await tx.order.update({ where: { id: pickingList.orderId }, data: { status: "COMPLETED" } });
      } else if (pickingList.status === "PENDING") {
        pickingList = await tx.pickingList.update({ where: { id }, data: { status: "IN_PROGRESS" } });
      }

      return { item: updatedItem, pickingList };
    });

    res.json(result);
  } catch (err) {
    if (err.status) {
      return res.status(err.status).json({ error: err.message });
    }
    throw err;
  }
});

module.exports = router;
