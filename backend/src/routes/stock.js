const express = require("express");
const prisma = require("../lib/prisma");
const { authRequired } = require("../middleware/auth");

const router = express.Router();

router.use(authRequired);

router.get("/", async (req, res) => {
  const stocks = await prisma.stock.findMany({
    include: { product: true, location: true },
    orderBy: { updatedAt: "desc" },
  });
  res.json(stocks);
});

router.post("/adjust", async (req, res) => {
  const { productId, locationId, quantity } = req.body;

  if (!productId || !locationId || typeof quantity !== "number") {
    return res.status(400).json({ error: "productId, locationId e quantity (number) são obrigatórios" });
  }

  try {
    const stock = await prisma.$transaction(async (tx) => {
      const current = await tx.stock.findUnique({
        where: { productId_locationId: { productId, locationId } },
      });

      const nextQuantity = (current?.quantity ?? 0) + quantity;
      if (nextQuantity < 0) {
        throw new Error("NEGATIVE_STOCK");
      }

      return tx.stock.upsert({
        where: { productId_locationId: { productId, locationId } },
        update: { quantity: nextQuantity },
        create: { productId, locationId, quantity: nextQuantity },
        include: { product: true, location: true },
      });
    });

    res.json(stock);
  } catch (err) {
    if (err.message === "NEGATIVE_STOCK") {
      return res.status(409).json({ error: "Ajuste resultaria em estoque negativo" });
    }
    throw err;
  }
});

module.exports = router;
