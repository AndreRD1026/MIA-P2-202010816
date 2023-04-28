import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Injectable({
  providedIn: 'root'
})
export class ProyectoService {

  // constructor() { }
  constructor(private httpClient: HttpClient) { }

  // postEntrada(entrada: string) {
  //   return this.httpClient.post("http://localhost:5000/analizar", { Cmd: entrada });
  // }

  postEntrada(entrada: string) {
    return this.httpClient.post("http://192.168.1.15:5000/analizar", { Cmd: entrada });
  }

  postLogin(id: string, user: string, pass: string) {
    const body = {
      id: id,
      user: user,
      pass: pass
    };
    return this.httpClient.post("http://localhost:5000/login", body);
  }

  postLogout(){
    let id = ""
    return this.httpClient.post("http://localhost:5000/logout", id);
  }

  // postLogout() {
  //   return this.httpClient.post("http://localhost:5000/logout")
  //     .pipe(
  //       map((response: any) => {
  //         return response.result;
  //       })
  //     );
  // }
}
