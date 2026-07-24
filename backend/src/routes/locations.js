const express = require("express");
const prisma = require("../lib/prisma");
const { authRequired } = require("../middleware/auth");

const router = express.Router();

router.use(authRequired);

router.get("/", async (req, res) => {
  const locations = await prisma.location.findMany({ orderBy: { code: "asc" } });
  res.json(locations);
});

router.post("/", async (req, res) => {
  const { code, description } = req.body;

  if (!code) {
    return res.status(400).json({ error: "code é obrigatório" });
  }

  const existing = await prisma.location.findUnique({ where: { code } });
  if (existing) {
    return res.status(409).json({ error: "Já existe uma localização com este código" });
  }

  const location = await prisma.location.create({ data: { code, description } });
  res.status(201).json(location);
});

module.exports = router;
