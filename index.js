const express = require('express');
const app = express();
const PORT = process.env.PORT || 3001;

// Başlangıç için veritabanı yerine hafızada basit bir kullanıcı listesi tutuyoruz.
const users = {
  '1001': { secret: 'pass123', realm: 'sentiric' },
  '1002': { secret: 'test456', realm: 'sentiric' },
};

app.get('/users/:username', (req, res) => {
  const username = req.params.username;
  const user = users[username];
  
  console.log(`[User Service] '${username}' için kullanıcı sorgusu alındı.`);

  if (user) {
    console.log(`--> Kullanıcı bulundu. Bilgiler gönderiliyor.`);
    res.status(200).json({
      username: username,
      ...user
    });
  } else {
    console.log(`--> Kullanıcı bulunamadı. 404 Not Found yanıtı gönderiliyor.`);
    res.status(404).json({ error: 'User not found' });
  }
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`✅ [User Service] Servis http://0.0.0.0:${PORT} adresinde dinlemede.`);
});