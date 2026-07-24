const express = require("express");
const prisma = require("../lib/prisma");
const { authRequired } = require("../middleware/auth");

const router = express.Router();

router.use(authRequired);

router.get("/", async (req, res) => {
  const orders = await prisma.order.findMany({
    include: { items: { include: { product: true } }, pickingList: true },
    orderBy: { createdAt: "desc" },
  });
  res.json(orders);
});

router.get("/:id", async (req, res) => {
  const order = await prisma.order.findUnique({
    where: { id: req.params.id },
    include: { items: { include: { product: true } }, pickingList: true },
  });

  if (!order) {
    return res.status(404).json({ error: "Pedido não encontrado" });
  }

  res.json(order);
});

router.post("/", async (req, res) => {
  const { orderNumber, items } = req.body;

  if (!orderNumber || !Array.isArray(items) || items.length === 0) {
    return res.status(400).json({ error: "orderNumber e items (não vazio) são obrigatórios" });
  }

  for (const item of items) {
    if (!item.productId || !item.quantity || item.quantity <= 0) {
      return res.status(400).json({ error: "Cada item precisa de productId e quantity > 0" });
    }
  }

  const existing = await prisma.order.findUnique({ where: { orderNumber } });
  if (existing) {
    return res.status(409).json({ error: "Já existe um pedido com este número" });
  }

  const order = await prisma.order.create({
    data: {
      orderNumber,
      items: {
        create: items.map((item) => ({ productId: item.productId, quantity: item.quantity })),
      },
    },
    include: { items: { include: { product: true } } },
  });

  res.status(201).json(order);
});

module.exports = router;
