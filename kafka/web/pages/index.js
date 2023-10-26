import Head from 'next/head'
import styles from '../styles/Home.module.css'
import { useEffect, useState } from 'react'

export default function Home() {
  const [message, setMessage] = useState("No Data Yet")
  useEffect(() => {
    const getData = async () => {
      const response = await  fetch("/api/data")
      const data = await response.json()
      setMessage(data.message)
    }
    const intervalId = setInterval(() => {
      getData()
    }, 1000 * 5) // in milliseconds
    return () => clearInterval(intervalId)

  }, [])

  return (
    <div className={styles.container}>
      <Head>
        <title>Confidential Containers on AKS</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>

      <main className={styles.main}>
        <h1 className={styles.title}>
          Welcome to <a href="https://github.com/microsoft/kata-containers">Confidential Containers on AKS!</a>
        </h1>

        <h2>
          {message.length > 50 ? "Encrypted" : "Decrypted"} Kafka Message:
        </h2>
        <p className={styles.description}>
        <code className={styles.code}>{message}</code>
        </p>
      </main>
    </div>
  )
}
