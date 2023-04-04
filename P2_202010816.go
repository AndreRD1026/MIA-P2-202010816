package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Partition = struct {
	part_status [100]byte
	part_type   [100]byte
	part_fit    [100]byte
	part_start  [100]byte
	part_size   [100]byte
	part_name   [100]byte
}

type MBR = struct {
	mbr_tamano         [100]byte
	mbr_fecha_creacion [100]byte
	mbr_dsk_signature  [100]byte
	dsk_fit            [100]byte
	// mbr_partition_1 [100]byte
	// mbr_partition_2 [100]byte
	// mbr_partition_3 [100]byte
	// mbr_partition_4 [100]byte
}

type ejemplo = struct {
	Id        [100]byte
	Nombre    [100]byte
	Direccion [100]byte
	Telefono  [100]byte
}

func main() {
	analizar()
}

func msg_error(err error) {
	fmt.Println("Error: ", err)
}

func analizar() {
	finalizar := false
	fmt.Println("MIA - Ejemplo 7, Analizador a Mano con Go (exit para salir...)")
	reader := bufio.NewReader(os.Stdin)
	//  Ciclo para lectura de multiples comandos
	for !finalizar {
		fmt.Print("<Ejemplo_7>: ")
		comando, _ := reader.ReadString('\n')
		if strings.Contains(comando, "exit") {
			finalizar = true
		} else {
			if comando != "" && comando != "exit\n" {
				//  Separacion de comando y parametros
				split_comando(comando)
			}
		}
	}
}

func split_comando(comando string) {
	var commandArray []string
	// Eliminacion de saltos de linea
	comando = strings.Replace(comando, "\n", "", 1)
	comando = strings.Replace(comando, "\r", "", 1)
	// Guardado de parametros
	if strings.Contains(comando, "mostrar") {
		commandArray = append(commandArray, comando)
	} else {
		commandArray = strings.Split(comando, " ")
	}
	// Ejecicion de comando leido
	ejecucion_comando(commandArray)
}

func ejecucion_comando(commandArray []string) {
	// Identificacion de comando y ejecucion
	data := strings.ToLower(commandArray[0])
	if data == "crear_disco" {
		crear_disco(commandArray)
	} else {
		fmt.Println("Comando ingresado no es valido")
	}
	//else if data == "escribir" {
	//	escribir(commandArray)
	//} else if data == "mostrar" {
	//	mostrar()
	//} else if data == "registrox" {
	//	registrox(commandArray)
	//}

}

func crear_disco(commandArray []string) {
	tamano := 0
	dimensional := ""
	tamano_archivo := 0
	limite := 0
	bloque := make([]byte, 1024)
	// Lectura de parametros del comando
	for i := 0; i < len(commandArray); i++ {
		data := strings.ToLower(commandArray[i])
		if strings.Contains(data, ">tamaño=") {
			strtam := strings.Replace(data, ">tamaño=", "", 1)
			strtam = strings.Replace(strtam, "\"", "", 2)
			strtam = strings.Replace(strtam, "\r", "", 1)
			tamano2, err := strconv.Atoi(strtam)
			tamano = tamano2
			if err != nil {
				msg_error(err)
			}
		} else if strings.Contains(data, ">dimensional=") {
			dimensional = strings.Replace(data, ">dimensional=", "", 1)
			dimensional = strings.Replace(dimensional, "\"", "", 2)
		}
	}
	// Calculo de tamaño del archivo
	if strings.Contains(dimensional, "k") {
		tamano_archivo = tamano
	} else if strings.Contains(dimensional, "m") {
		tamano_archivo = tamano * 1024
	} else if strings.Contains(dimensional, "g") {
		tamano_archivo = tamano * 1024 * 1024
	}
	// Preparacion del bloque a escribir en archivo
	for j := 0; j < 1024; j++ {
		bloque[j] = 0
	}
	// Creacion, escritura y cierre de archivo
	disco, err := os.Create("Ejemplo7.dsk")
	if err != nil {
		msg_error(err)
	}
	for limite < tamano_archivo {
		_, err := disco.Write(bloque)
		if err != nil {
			msg_error(err)
		}
		limite++
	}
	disco.Close()
	// Resumen de accion realizada
	fmt.Print("Creacion de Disco:")
	fmt.Print(" Tamaño: ")
	fmt.Print(tamano)
	fmt.Print(" Dimensional: ")
	fmt.Println(dimensional)
}

func struct_to_bytes(p interface{}) []byte {
	// Codificacion de Struct a []Bytes
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(p)
	if err != nil && err != io.EOF {
		msg_error(err)
	}
	return buf.Bytes()
}

func bytes_to_struct(s []byte) ejemplo {
	// Decodificacion de [] Bytes a Struct ejemplo
	p := ejemplo{}
	dec := gob.NewDecoder(bytes.NewReader(s))
	err := dec.Decode(&p)
	if err != nil && err != io.EOF {
		msg_error(err)
	}
	return p
}
