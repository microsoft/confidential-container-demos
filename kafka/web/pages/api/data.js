// Next.js API route support: https://nextjs.org/docs/api-routes/introduction

export default async function data(req, res) {
  // fetch data from kafka consumer service and return response
    const response = await fetch("http://" + process.env.CONSUMER_SERVICE_HOST + ":" + process.env.CONSUMER_SERVICE_PORT+ "/")
    const data = await response.text()
    res.status(200).json({ message: data })
  }
